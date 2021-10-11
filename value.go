package gofactor

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"log"
)

func SeparateValues(input []byte) ([]byte, error) {
	f, err := parse(input)
	if err != nil {
		return nil, err
	}

	cc := &valueCleaner{formatter: f}
	cc.separateValDecls()
	output, err := format.Source(cc.writer.Bytes())
	if err != nil {
		output = cc.writer.Bytes()
		log.Printf("Unexpected formatting error: %v", err)
	}
	return output, err
}

type valueCleaner struct {
	*formatter
}

func (cc *valueCleaner) separateValDecl(decl *ast.GenDecl) {
	pos := decl.Pos()
	// only refactor parenthesized decalarations
	if decl.Lparen.IsValid() {
		lastType := ""
		var prev *ast.ValueSpec
		for _, spec := range decl.Specs {
			vs := spec.(*ast.ValueSpec)
			if len(vs.Names) < 2 {
				if lastType == "" {
					lastType = cc.typStr(vs)
				}

				// check if the next spec type is different than
				// the previous, and if so, close the block
				// and start a new one
				if vs.Type != nil && lastType != cc.typStr(vs) {
					// make sure to capture the comment
					end := prev.End()
					if prev.Comment != nil {
						end = prev.Comment.End()
					}

					// write out the previous spec
					cc.writePos(pos, end)

					start := vs.Pos()
					if vs.Doc != nil {
						start = vs.Doc.Pos()
					}

					// end the const block and start a new one
					// with the next spec
					cc.write(fmt.Sprintf("\n)\n\n%s (\n", decl.Tok))
					lastType = cc.typStr(vs)
					pos = start
				}
			}
			prev = vs
		}

		if pos != decl.Pos() {
			cc.writePos(pos, decl.End())
		}
	}

	if pos == decl.Pos() {
		cc.writePos(decl.Pos(), decl.End())
	}
}

func (cc *valueCleaner) separateValDecls() {
	pos := cc.f.Pos()
	for _, decl := range cc.f.Decls {
		if d, ok := decl.(*ast.GenDecl); ok {
			cc.writePos(pos, d.Pos())
			if d.Tok == token.CONST || d.Tok == token.VAR {
				cc.separateValDecl(d)
				pos = decl.End()
			} else {
				pos = decl.Pos()
			}
		} else {
			pos = decl.End()
		}
	}
}