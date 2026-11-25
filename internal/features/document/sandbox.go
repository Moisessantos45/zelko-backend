package document

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const (
	MaxSourceBytes = 1 << 20 // 1 MiB
	CompileTimeout = 60 * time.Second
	maxLogBytes    = 256 << 10
)

type rule struct {
	re  *regexp.Regexp
	msg string
}

func r(p, msg string) rule { return rule{regexp.MustCompile(p), msg} }

// Reglas de rechazo. Son una primera barrera: la defensa real es la
// ejecución restringida (sin shell-escape, timeout, entorno limpio y bwrap).
var latexRules = []rule{
	r(`\\write\s*18`, "shell escape (\\write18) no permitido"),
	r(`\\immediate\s*\\write`, "\\immediate\\write no permitido"),
	r(`\\openout`, "\\openout no permitido"),
	r(`\\openin`, "\\openin no permitido"),
	r(`\\read\b`, "\\read no permitido"),
	r(`\\(directlua|latelua|luaexec)`, "código Lua no permitido"),
	r(`\\begin\s*\{\s*luacode\*?\s*\}`, "luacode no permitido"),
	r(`\{\s*(shellesc|minted|pythontex|sagetex|gnuplottex|write18)\s*\}`, "paquete que ejecuta comandos no permitido"),
	r(`\\(input|include|includeonly|usepackage|RequirePackage|documentclass|lstinputlisting|verbatiminput|includegraphics|includepdf)\s*(\[[^\]]*\])?\s*\{\s*[|/~]`, "ruta absoluta o pipe no permitido"),
	r(`\\(input|include|lstinputlisting|verbatiminput|includegraphics|includepdf)\s*(\[[^\]]*\])?\s*\{[^}]*\.\.`, "ruta con '..' no permitida"),
	r(`\\input\s*\|`, "pipe en \\input no permitido"),
	r(`\\catcode`, "\\catcode no permitido"),
	r(`\\csname\s*(write|immediate|openout|read|input)`, "construcción sospechosa no permitida"),
}

var typstRules = []rule{
	r(`\bplugin\s*\(`, "plugins WASM no permitidos"),
	r(`\bread\s*\(`, "read() no permitido"),
	r(`\b(image|include|csv|json|yaml|toml|xml|cbor|bibliography)\s*\(\s*"[^"]*(\.\.|^/)`, "ruta con '..' no permitida"),
	r(`\b(image|include|csv|json|yaml|toml|xml|cbor|bibliography|read)\s*\(\s*"/`, "ruta absoluta no permitida"),
	r(`@local/`, "paquetes locales no permitidos"),
}

// ValidateSource rechaza contenido excesivo o con construcciones peligrosas.
func ValidateSource(engine, content string) error {
	if len(content) > MaxSourceBytes {
		return fmt.Errorf("el documento excede %d bytes", MaxSourceBytes)
	}
	if strings.ContainsRune(content, 0) {
		return errors.New("el documento contiene bytes nulos")
	}

	var rules []rule
	switch engine {
	case "latex":
		rules = latexRules
	case "typst":
		rules = typstRules
	default:
		return fmt.Errorf("motor no soportado: %s", engine)
	}

	for _, ru := range rules {
		if ru.re.MatchString(content) {
			return errors.New(ru.msg)
		}
	}

	if engine == "typst" {
		return validateTypstPackages(content)
	}
	return nil
}

var typstPkgRef = regexp.MustCompile(`@([A-Za-z0-9_-]+)/([A-Za-z0-9._-]*)(?::([A-Za-z0-9._-]+))?`)

// typstPackageCache devuelve el directorio de paquetes de Typst ya instalados.
func typstPackageCache() string {
	if d := os.Getenv("TYPST_PACKAGE_CACHE_PATH"); d != "" {
		return d
	}
	if d, err := os.UserCacheDir(); err == nil {
		return filepath.Join(d, "typst", "packages")
	}
	return ""
}

// Solo se permiten paquetes @preview que ya estén instalados en la caché
// (el sandbox no tiene red, así que nunca se descarga nada).
func validateTypstPackages(content string) error {
	for _, m := range typstPkgRef.FindAllStringSubmatch(content, -1) {
		ns, name, ver := m[1], m[2], m[3]
		if ns != "preview" {
			continue // p. ej. correos "user@example.com" no cumplen el patrón de import
		}
		if name == "" || ver == "" {
			return errors.New("paquete de Typst inválido (usa @preview/nombre:versión)")
		}
		dir := filepath.Join(typstPackageCache(), "preview", name, ver)
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			return fmt.Errorf("paquete @preview/%s:%s no está instalado en el servidor", name, ver)
		}
	}
	return nil
}

type limitedBuffer struct {
	buf bytes.Buffer
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	if rem := maxLogBytes - l.buf.Len(); rem > 0 {
		if len(p) > rem {
			l.buf.Write(p[:rem])
		} else {
			l.buf.Write(p)
		}
	}
	return len(p), nil // descarta el exceso sin bloquear el proceso
}

// RunCompiler ejecuta el compilador de forma restringida y devuelve su log.
func RunCompiler(engine, jobDir, srcName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), CompileTimeout)
	defer cancel()

	var args []string
	switch engine {
	case "latex":
		args = []string{
			"latexmk", "-pdf", "-norc", "-interaction=nonstopmode",
			"-halt-on-error", "-no-shell-escape", srcName,
		}
	case "typst":
		args = []string{"typst", "compile", "--root", jobDir, srcName, "main.pdf"}
	default:
		return "", fmt.Errorf("motor no soportado: %s", engine)
	}

	env := []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOME=" + jobDir,
		"LANG=C.UTF-8",
		"openin_any=p",  // TeX solo lee archivos no ocultos / relativos
		"openout_any=p", // TeX solo escribe en el directorio actual
		"shell_escape=f",
	}

	// Aislamiento con bubblewrap si está instalado.
	if bwrap, err := exec.LookPath("bwrap"); err == nil {
		bw := []string{
			bwrap,
			"--ro-bind", "/", "/",
			"--dev", "/dev",
			"--proc", "/proc",
			"--tmpfs", "/tmp",
			"--tmpfs", "/home",
			"--tmpfs", "/root",
		}
		if engine == "typst" {
			// Paquetes y fuentes del usuario, solo lectura.
			if pk := typstPackageCache(); pk != "" {
				if _, err := os.Stat(pk); err == nil {
					bw = append(bw, "--ro-bind", pk, pk)
					env = append(env, "TYPST_PACKAGE_CACHE_PATH="+pk)
				}
			}
			if h, err := os.UserHomeDir(); err == nil {
				fonts := filepath.Join(h, ".local", "share", "fonts")
				if _, err := os.Stat(fonts); err == nil {
					bw = append(bw, "--ro-bind", fonts, fonts)
					env = append(env, "TYPST_FONT_PATHS="+fonts)
				}
			}
		}
		args = append(append(bw, []string{
			"--bind", jobDir, jobDir,
			"--unshare-all",
			"--die-with-parent",
			"--new-session",
			"--chdir", jobDir,
		}...), args...)
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = jobDir
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 2 * time.Second

	var out limitedBuffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = fmt.Errorf("tiempo de compilación excedido (%s)", CompileTimeout)
	}
	return out.buf.String(), err
}
