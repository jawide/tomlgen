package main

import (
	"bytes"
	"flag"
	"fmt"
	"gitee.com/jawide/toml2go"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"
)

var (
	typeSet = flag.NewFlagSet("name", flag.ContinueOnError)
	initSet = flag.NewFlagSet("type", flag.ContinueOnError)

	outputFilePath  = flag.String("output", "", "output file path")
	outputFilePath2 = flag.String("o", "", "output file path")
	structName      = flag.String("name", "", "specifies the name of the generated struct")
	structName2     = flag.String("n", "Config", "specifies the name of the generated struct")
	varName         = flag.String("var", "", "specifies the name of the var")
	varName2        = flag.String("v", "Cfg", "specifies the name of the var")
	packageName     = flag.String("package", "", "specifies the name of the package")
	packageName2    = flag.String("p", "main", "specifies the name of the package")
	configFilePath  = flag.String("config", "", "specifies the file used to load the configuration")
	configFilePath2 = flag.String("c", "", "specifies the file used to load the configuration")

	structNameForType  = typeSet.String("name", "", "specifies the name of the generated struct")
	structNameForType2 = typeSet.String("n", "Config", "specifies the name of the generated struct")
	varNameForInit     = initSet.String("name", "", "specifies the name of the var")
	varNameForInit2    = initSet.String("n", "Cfg", "specifies the name of the var")
)

func Usage() {
	_, _ = fmt.Fprintf(os.Stderr, "Description:\n")
	_, _ = fmt.Fprintf(os.Stderr, "  Parsing configFile generates go statement,\n")
	_, _ = fmt.Fprintf(os.Stderr, "Usage:\n")
	_, _ = fmt.Fprintf(os.Stderr, "  tomlgen [command] [options] tomlPath\n")
	_, _ = fmt.Fprintf(os.Stderr, "Examples:\n")
	_, _ = fmt.Fprintf(os.Stderr, "  tomlgen -o 'config.go' 'config.toml'\n")
	_, _ = fmt.Fprintf(os.Stderr, "  go generate tomlgen -o 'config.go' type 'config.toml'\n")
	_, _ = fmt.Fprintf(os.Stderr, "  go generate tomlgen -o 'config.go' init 'config.toml'\n")
	_, _ = fmt.Fprintf(os.Stderr, "Commands:\n")
	_, _ = fmt.Fprintf(os.Stderr, "  type\tgenerate struct from toml file\n")
	_, _ = fmt.Fprintf(os.Stderr, "  init\tgenerate init func from toml file\n")
	_, _ = fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
	_, _ = fmt.Fprintf(os.Stderr, "  type:\n")
	typeSet.PrintDefaults()
	_, _ = fmt.Fprintf(os.Stderr, "  init:\n")
	initSet.PrintDefaults()
	_, _ = fmt.Fprintf(os.Stderr, "For more information, see:\n")
	_, _ = fmt.Fprintf(os.Stderr, "  https://gitee.com/jawide/tomlgen\n")
}

type configTemplate struct {
	Package string
	Struct  string
	Var     string
	Type    string
	File    string
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("tomlgen: ")
	flag.Usage = Usage
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		Usage()
		return
	}
	cmd := args[0]
	var fs *token.FileSet
	var file *dst.File
	initStr := `func init() {
		file, err := ioutil.ReadFile("%s")
		if err != nil {
			panic(err)
		}
		err = toml.Unmarshal(file, &%s)
		if err != nil {
			panic(err)
		}
		}`
	templateStr := `package {{.Package}}
		import (
			"github.com/pelletier/go-toml"
			"io/ioutil"
		)
		
		{{.Struct}}
		
		var {{.Var}} {{.Type}}
		
		func init() {
			file, err := ioutil.ReadFile("{{.File}}")
			if err != nil {
				panic(err)
			}
			err = toml.Unmarshal(file, &{{.Var}})
			if err != nil {
				panic(err)
			}
		}`
	switch cmd {
	default:
		processFlag()
		structStr := genStructByToml(args[0], *structName)
		t := template.New("config")
		template.Must(t.Parse(templateStr))
		if *configFilePath != "" {
			args[0] = *configFilePath
		}
		var wr bytes.Buffer
		err := t.Execute(&wr, configTemplate{
			Package: *packageName,
			Struct:  structStr,
			Var:     *varName,
			Type:    *structName,
			File:    args[0],
		})
		if err != nil {
			log.Fatal(err)
		}
		source, err := format.Source(wr.Bytes())
		if err != nil {
			log.Fatal(err)
		}
		if *outputFilePath != "" {
			f, err := os.Create(*outputFilePath)
			if err != nil {
				log.Fatal(err)
			}
			_, err = f.Write(source)
			if err != nil {
				log.Fatal(err)
			}
			err = f.Close()
			if err != nil {
				log.Fatal(err)
			}
		} else {
			fmt.Println(string(source))
		}
		break
	case "type":
		err := typeSet.Parse(args[1:])
		processFlag()
		if err != nil {
			log.Fatal(err)
		}
		structStr := genStructByToml(typeSet.Arg(0), *structNameForType)
		structNode := newStructNode(structStr)
		if fs == nil {
			fs = token.NewFileSet()
		}
		if file == nil {
			file, _ = decorator.ParseFile(fs, *outputFilePath, nil, parser.AllErrors|parser.ParseComments)
		}
		dst.Inspect(file, func(node dst.Node) bool {
			switch n := node.(type) {
			case *dst.GenDecl:
				if len(n.Decs.NodeDecs.Start) > 0 {
					s := n.Decs.NodeDecs.Start[0]
					if regexp.MustCompile(`//go:generate\s+tomlgen\s+.*type.*`).MatchString(s) {
						n.Specs[0] = structNode
					}
				}
				break
			}
			return true
		})
		saveDstFile(file)
		break
	case "init":
		err := initSet.Parse(args[1:])
		processFlag()
		if err != nil {
			log.Fatal(err)
		}
		if fs == nil {
			fs = token.NewFileSet()
		}
		if file == nil {
			file, _ = decorator.ParseFile(fs, *outputFilePath, nil, parser.AllErrors|parser.ParseComments)
		}
		dst.Inspect(file, func(node dst.Node) bool {
			switch n := node.(type) {
			case *dst.FuncDecl:
				if len(n.Decs.NodeDecs.Start) > 0 {
					s := n.Decs.NodeDecs.Start[0]
					if regexp.MustCompile(`//go:generate\s+tomlgen\s+.*init.*`).MatchString(s) {
						n.Body = newFuncBodyNode(fmt.Sprintf(initStr, initSet.Arg(0), *varNameForInit))
					}
				}
				break
			}
			return true
		})
		saveDstFile(file)
		break
	}
}

func saveDstFile(file *dst.File) {
	f, err := os.OpenFile(*outputFilePath, os.O_WRONLY, 2)
	if err != nil {
		log.Fatal(err)
	}
	err = f.Truncate(0)
	if err != nil {
		log.Fatal(err)
	}
	err = decorator.Fprint(f, file)
	if err != nil {
		log.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func newStructNode(structStr string) *dst.TypeSpec {
	mockFile, err := decorator.Parse("package main\n" + structStr)
	if err != nil {
		panic(err)
	}
	return mockFile.Decls[0].(*dst.GenDecl).Specs[0].(*dst.TypeSpec)
}

func newFuncBodyNode(funcStr string) *dst.BlockStmt {
	mockFile, err := decorator.Parse("package main\n" + funcStr)
	if err != nil {
		log.Fatal(err)
	}
	return mockFile.Decls[0].(*dst.FuncDecl).Body
}

func genStructByToml(filepath, structName string) string {
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatal(err)
	}
	goStatement, err := toml2go.Toml2Go(string(file), true)
	if err != nil {
		log.Fatal(err)
	}
	index := strings.Index(goStatement, "\n")
	return strings.Replace(goStatement[:index], "AutoGenerated", structName, 1) + goStatement[index:]
}

func processFlag() {
	if *outputFilePath == "" {
		outputFilePath = outputFilePath2
	}
	if *structName == "" {
		structName = structName2
	}
	if *varName == "" {
		varName = varName2
	}
	if *structNameForType == "" {
		structNameForType = structNameForType2
	}
	if *varNameForInit == "" {
		varNameForInit = varNameForInit2
	}
	if *packageName == "" {
		packageName = packageName2
	}
	if *configFilePath == "" {
		configFilePath = configFilePath2
	}
}
