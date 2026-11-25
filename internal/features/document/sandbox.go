package document

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
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
	r(`@(preview|local)/`, "paquetes externos no permitidos"),
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

	// Aislamiento con bubblewrap si está instalado.
	if bwrap, err := exec.LookPath("bwrap"); err == nil {
		args = append([]string{
			bwrap,
			"--ro-bind", "/", "/",
			"--dev", "/dev",
			"--proc", "/proc",
			"--tmpfs", "/tmp",
			"--tmpfs", "/home",
			"--tmpfs", "/root",
			"--bind", jobDir, jobDir,
			"--unshare-all",
			"--die-with-parent",
			"--new-session",
			"--chdir", jobDir,
		}, args...)
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = jobDir
	cmd.Env = []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOME=" + jobDir,
		"LANG=C.UTF-8",
		"openin_any=p",  // TeX solo lee archivos no ocultos / relativos
		"openout_any=p", // TeX solo escribe en el directorio actual
		"shell_escape=f",
	}
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
