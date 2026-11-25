package document

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate(t *testing.T) {
	bad := map[string]string{
		"latex": `\immediate\write18{rm -rf /}`,
	}
	for e, s := range bad {
		if ValidateSource(e, s) == nil {
			t.Fatalf("debió rechazar %s", s)
		}
	}
	for _, s := range []string{`\input{/etc/passwd}`, `\input{../x}`, `\usepackage{shellesc}`, `\directlua{os.execute("id")}`} {
		if ValidateSource("latex", s) == nil {
			t.Fatalf("debió rechazar %s", s)
		}
	}
	for _, s := range []string{`#read("/etc/passwd")`, `#plugin("x.wasm")`, `#image("../a.png")`} {
		if ValidateSource("typst", s) == nil {
			t.Fatalf("debió rechazar %s", s)
		}
	}
	if ValidateSource("latex", `\documentclass{article}\begin{document}Hola\end{document}`) != nil {
		t.Fatal("falso positivo latex")
	}
	if ValidateSource("typst", "= Hola\n#lorem(10)") != nil {
		t.Fatal("falso positivo typst")
	}
}

func TestCompileReal(t *testing.T) {
	for eng, src := range map[string]struct{ n, c string }{
		"latex": {"main.tex", `\documentclass{article}\begin{document}Hola\end{document}`},
		"typst": {"main.typ", "= Hola\n"},
	} {
		d := t.TempDir()
		os.WriteFile(filepath.Join(d, src.n), []byte(src.c), 0600)
		log, err := RunCompiler(eng, d, src.n)
		if err != nil {
			t.Fatalf("%s: %v\n%s", eng, err, log)
		}
		if _, err := os.Stat(filepath.Join(d, "main.pdf")); err != nil {
			t.Fatalf("%s sin pdf\n%s", eng, log)
		}
	}
}
