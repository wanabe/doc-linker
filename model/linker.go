package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

type Linker struct {
	dir     string
	nodeMap map[string]*Node
	rule    *Rule
}

var (
	tmplMmd    = template.Must(template.New("tmpl").Parse("  click {{.Name}} \"{{.Link}}\"\n"))
	regName    = regexp.MustCompile("^\n*# +(.*)")
	regPara    = regexp.MustCompile("## +(.*)")
	regCode    = regexp.MustCompile("(?m)^```(?:.*\n)*?^```")
	regPreType = regexp.MustCompile("``` *(.*) *")
	regMmdNode = regexp.MustCompile("(?m)^ *(?:(.*?) *--> (.*)|subgraph +(.*)) *$")
	regClicks  = regexp.MustCompile("\n+(?:\n *click .*)+\n*")
)

func NewLinker(dir string) (*Linker, error) {
	l := &Linker{dir: dir}
	err := l.normalizeDir()
	if err != nil {
		return nil, err
	}

	err = l.readLinkRule(filepath.Join(l.dir, "link.json"))
	if err != nil {
		return nil, err
	}

	err = l.buildNodeMap()
	if err != nil {
		return nil, err
	}

	return l, nil
}

func (l *Linker) normalizeDir() error {
	dir, err := filepath.Abs(l.dir)
	if err != nil {
		return err
	}
	l.dir = dir
	return nil
}

func (l *Linker) readLinkRule(path string) error {
	rule := Rule{}
	b, err := os.ReadFile(path)
	json.Unmarshal(b, &rule)
	if rule.Rules.Mermaid != nil {
		if rule.Rules.Mermaid.Anchor == "" || rule.Rules.Mermaid.File == "" {
			return errors.New("invalid mermaid rule: missing anchor and/or file")
		}

		rule.Rules.Mermaid.anchorTmpl, err = template.New("anchor").Parse(rule.Rules.Mermaid.Anchor)
		if err != nil {
			return err
		}
		rule.Rules.Mermaid.fileTmpl, err = template.New("file").Parse(rule.Rules.Mermaid.File)
		if err != nil {
			return err
		}
	}
	l.rule = &rule
	return nil
}

func (l *Linker) walk(f func(string) error) error {
	return filepath.Walk(l.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fInfo, err := os.Stat(path); err != nil || fInfo.IsDir() {
			return err
		}

		return f(path)
	})
}

func (l *Linker) buildNodeMap() error {
	l.nodeMap = map[string]*Node{}
	err := l.walk(func(path string) error {
		rel, err := filepath.Rel(l.dir, path)
		if err != nil {
			return err
		}
		doc := &Node{FilePath: path, Path: rel, pMap: map[string]*Node{}}
		l.nodeMap[path] = doc
		l.nodeMap[rel] = doc

		if filepath.Ext(path) != ".md" {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		iss := regName.FindAllSubmatch(b, -1)
		if len(iss) > 0 {
			if len(iss[0]) > 1 {
				key := string(iss[0][1])
				l.nodeMap[key] = doc
			}
		}

		iss = regPara.FindAllSubmatch(b, -1)
		for _, is := range iss {
			if len(is) > 1 {
				key := string(is[1])
				d := *doc
				d.Name = key
				doc.pMap[key] = &d
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (l *Linker) LinkDocs() error {
	return l.walk(func(path string) error {
		if filepath.Ext(path) != ".md" {
			return nil
		}
		doc := l.nodeMap[path]

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		b = regCode.ReplaceAllFunc(b, func(b []byte) []byte {
			bs := regPreType.FindSubmatch(b)
			typ := func() string {
				if len(bs) <= 1 {
					return ""
				}
				return string(bs[1])
			}()
			switch typ {
			case "mermaid":
				if l.rule.Rules.Mermaid == nil {
					return b
				}
				nMap := map[string]bool{}
				bMap := map[string]bool{}
				iss := regMmdNode.FindAllSubmatch(b, -1)
				for _, is := range iss {
					if len(is) <= 1 {
						continue
					}
					for i, b := range is[1:] {
						if len(b) == 0 {
							continue
						}
						k := string(b)
						if i == 2 {
							bMap[k] = true
							delete(nMap, k)
							continue
						}
						if !bMap[k] && (l.nodeMap[k] != nil || doc.pMap[k] != nil) {
							nMap[k] = true
						}
					}
				}
				names := []string{}
				for n := range nMap {
					names = append(names, n)
				}
				sort.Strings(names)

				b = regClicks.ReplaceAll(b, []byte{'\n'})
				builder := bytes.NewBufferString("")
				builder.Write(b[:len(b)-3])

				if len(nMap) > 0 {
					builder.WriteByte('\n')
					for _, n := range names {
						d := doc.pMap[n]
						if d == nil {
							d = l.nodeMap[n]
						}
						v := map[string]string{
							"Path": d.Path,
							"Name": d.Name,
						}
						for vn, vs := range l.rule.Constants {
							v[vn] = vs
						}

						linkBuilder := bytes.NewBuffer([]byte{})
						if d.Name == "" {
							l.rule.Rules.Mermaid.fileTmpl.Execute(linkBuilder, v)
						} else {
							l.rule.Rules.Mermaid.anchorTmpl.Execute(linkBuilder, v)
						}
						tmplMmd.Execute(builder, map[string]string{"Name": n, "Link": linkBuilder.String()})
					}
				}
				builder.Write([]byte("```"))
				return builder.Bytes()
			}

			return b
		})
		os.WriteFile(path, b, 664)
		return nil
	})
}
