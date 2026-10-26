package riskmanager_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

var domainWords = map[string]bool{
	"ai": true, "llm": true, "prompt": true, "rag": true, "dlp": true,
	"leak": true, "leakage": true, "copilot": true, "chatbot": true,
	"agent": true, "enterprise": true, "tenant": true, "vendor": true,
	"food": true, "farm": true, "crop": true, "produce": true,
	"harvest": true, "allergen": true, "contamination": true,
	"pipeline": true, "valve": true, "vehicle": true, "patient": true,
	"hospital": true, "worker": true, "factory": true,
}

func tokensOf(s string) []string {
	var tokens []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			tokens = append(tokens, strings.ToLower(string(current)))
			current = nil
		}
	}
	runes := []rune(s)
	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		if len(current) > 0 {
			prev := runes[i-1]
			startsWord := unicode.IsUpper(r) && unicode.IsLower(prev)
			endsAcronym := unicode.IsUpper(r) && unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1])
			switchesKind := unicode.IsDigit(r) != unicode.IsDigit(prev)
			if startsWord || endsAcronym || switchesKind {
				flush()
			}
		}
		current = append(current, r)
	}
	flush()
	return tokens
}

func carriesDomainWord(s string) (string, bool) {
	for _, token := range tokensOf(s) {
		if domainWords[token] {
			return token, true
		}
	}
	return "", false
}

func TestTheTokenizerReadsWholeWordsOnly(t *testing.T) {
	for _, c := range []struct {
		in   string
		want bool
	}{
		{"storage", false},
		{"ErrInvalid", false},
		{"RiskID", false},
		{"RAGIndex", true},
		{"promptInjection", true},
		{"leak_path", true},
		{"/risks/{riskID}/pathways", false},
		{"riskmanager.risk.create", false},
		{"AIGateway", true},
		{"retained", false},
	} {
		if _, got := carriesDomainWord(c.in); got != c.want {
			t.Errorf("carriesDomainWord(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestNoDomainWordsInIdentifiers(t *testing.T) {
	for _, dir := range []string{"models", "routes", "."} {
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", dir, err)
		}
		seen := map[string]bool{}
		for _, pkg := range pkgs {
			for path, file := range pkg.Files {
				ast.Inspect(file, func(n ast.Node) bool {
					id, ok := n.(*ast.Ident)
					if !ok {
						return true
					}
					w, bad := carriesDomainWord(id.Name)
					if !bad {
						return true
					}
					key := filepath.Base(path) + ":" + id.Name
					if seen[key] {
						return true
					}
					seen[key] = true
					t.Errorf("%s: identifier %q carries the domain word %q — it belongs in a risk's document or in a domain's own spec",
						filepath.Base(path), id.Name, w)
					return true
				})
			}
		}
	}
}

func TestNoDomainWordsInStoredVocabularies(t *testing.T) {
	for at, values := range heldVocabularies() {
		for _, v := range values {
			if w, bad := carriesDomainWord(v); bad {
				t.Errorf("stored value %s = %q carries the domain word %q — a value outlives a rename", at, v, w)
			}
		}
	}
}

func TestNoDomainWordsInTheBlueprintsRolesFieldsAndStoredValues(t *testing.T) {
	b, err := riskmanager.Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}
	report := func(kind, name string) {
		if w, bad := carriesDomainWord(name); bad {
			t.Errorf("blueprint %s %q carries the domain word %q — it belongs in the document or in a domain's own spec", kind, name, w)
		}
	}
	for _, o := range b.Objects {
		report("role", o.Role)
		if o.Name != "" || o.Table != "" {
			t.Errorf("blueprint role %q names a table, which is the domain's to choose", o.Role)
		}
		for _, f := range o.Fields {
			report("field", o.Role+"."+f.Name)
			for _, v := range f.Values {
				report("stored value", o.Role+"."+f.Name+"="+v)
			}
		}
		for _, idx := range o.Indexes {
			report("index", o.Role+"."+idx.Name)
			if idx.Path != nil {
				report("index path", idx.Path.Path)
			}
		}
	}
}

func TestNoDomainWordsInTheDeclaredRouteTable(t *testing.T) {
	var declared struct {
		Operations map[string]struct {
			Path   string `json:"path"`
			Action string `json:"action"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(riskmanager.Operations(), &declared); err != nil {
		t.Fatalf("parse operations: %v", err)
	}
	if len(declared.Operations) == 0 {
		t.Fatal("the operations file declares nothing")
	}
	for name, op := range declared.Operations {
		for field, value := range map[string]string{"path": op.Path, "action": op.Action} {
			if w, bad := carriesDomainWord(value); bad {
				t.Errorf("operation %s: the %s %q carries the domain word %q — an address outlives a rename too", name, field, value, w)
			}
		}
	}
}
