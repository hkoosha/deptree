package cmd

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/vc60er/deptree/internal/moduleinfo"
	"github.com/vc60er/deptree/internal/tree"
	"github.com/vc60er/deptree/internal/verbose"
)

// Execute starts the main code
// - get all upgradable modules: "go list -u -m -json all" and only those with newer version
// - filter list of "go mod graph" (all children with parent) to all upradeable children
// - print all parents needs to upgrade for usage of its (direct) upgradable children
// - colored output
func Execute() {
	// TODO:
	// - check go version
	showAll := flag.Bool("a", true, "show all dependencies, also without upgrade and point out duplicated children")
	colored := flag.Bool("c", false, "upgrade candidates will be marked yellow")
	depth := flag.Int("d", 5, "max depth of dependencies")
	showDroppedChild := flag.Bool("f", true, "force show of each occurrence of a child branch in tree (can cause hang)")
	visualizeTrimmed := flag.Bool("t", true, "visualize trimmed tree by '└─...'")
	printJSON := flag.Bool("json", false, "print JSON instead of tree")
	graphFile := flag.String("graph", "grapphfile.txt", "path to file created e.g. by 'go mod graph > grapphfile.txt'")
	upgradeFile := flag.String("upgrade", "upgradefile.txt", "path to file created e.g. by 'go list -u -m -json all > upgradefile.txt'")
	verboseLevel := flag.Int("v", 0, "be more verbose")
	flag.Parse()

	root := ""
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	v := verbose.NewVerbose(*verboseLevel)

	info := moduleinfo.NewInfo(v)
	info.Fill(getUpgradeContent(*upgradeFile, v))
	v.Log1f("fill with upgrade content done")

	tree := tree.NewTree(root, v, *depth, *showDroppedChild, *visualizeTrimmed, *showAll, *colored, *info)
	file := getGraphFile(*graphFile, v)
	defer file.Close()
	tree.Fill(file)
	v.Log1f("fill with graph content done")

	info.Adjust()
	v.Log1f("content adjusted")

	tree.Print(*printJSON)
	v.Log1f("finished")
}

// getUpgradeContent gets the JSON content from go list call or upgrade file
func getUpgradeContent(upgradeFile string, verbose verbose.Verbose) []byte {
	var goListCallJSONContent []byte
	if len(upgradeFile) == 0 {
		fmt.Println("call 'go list -u -m -json all', be patient...")
		var outbuf, errbuf bytes.Buffer
		cmd := exec.Command("go", "list", "-u", "-m", "-json", "all")
		cmd.Stdout = &outbuf
		cmd.Stderr = &errbuf
		if err := cmd.Run(); err != nil {
			log.Fatalf("%v, %s", err, errbuf.String())
		}
		goListCallJSONContent = outbuf.Bytes()
	} else {
		var err error
		if upgradeFile, err = filepath.Abs(upgradeFile); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("use upgrade file %s\n", upgradeFile)
		if goListCallJSONContent, err = ioutil.ReadFile(upgradeFile); err != nil {
			log.Fatal(err)
		}
	}

	verbose.Log1f("upgrade content retrieved")
	return goListCallJSONContent
}

// getGraphFile gets the file handle to access content from STDIN or graph file
func getGraphFile(graphFile string, verbose verbose.Verbose) *os.File {
	var err error
	var file *os.File
	if len(graphFile) == 0 {
		file = os.Stdin
	} else {
		if graphFile, err = filepath.Abs(graphFile); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("use graph file %s\n", graphFile)
		file, err = os.Open(graphFile)
		if err != nil {
			log.Fatal(err)
		}
	}
	verbose.Log1f("graph content retrieved")
	return file
}
