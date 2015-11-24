package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/github/git-lfs/lfs"
	"github.com/github/git-lfs/vendor/_nuts/github.com/spf13/cobra"
)

type ServerTest struct {
	Name string
	F    func(endp lfs.Endpoint, oidsExist, oidsMissing []string) error
}

var (
	RootCmd = &cobra.Command{
		Use:   "git-lfs-test-server-api [--url=<apiurl> | --clone=<cloneurl>] [<oid-exists-file> <oid-missing-file>]",
		Short: "Test a Git LFS API server for compliance",
		Run:   testServerApi,
	}
	apiUrl   string
	cloneUrl string

	tests []ServerTest
)

func main() {
	RootCmd.Execute()
}

func testServerApi(cmd *cobra.Command, args []string) {

	if (len(apiUrl) == 0 && len(cloneUrl) == 0) ||
		(len(apiUrl) != 0 && len(cloneUrl) != 0) {
		exit("Must supply either --url or --clone (and not both)")
	}

	if len(args) != 0 && len(args) != 2 {
		exit("Must supply either no file arguments or both the exists AND missing file")
	}

	var endp lfs.Endpoint
	if len(cloneUrl) > 0 {
		endp = lfs.NewEndpointFromCloneURL(cloneUrl)
	} else {
		endp = lfs.NewEndpoint(apiUrl)
	}

	var oidsExist, oidsMissing []string
	if len(args) >= 2 {
		fmt.Printf("Reading test data from files (no server content changes)\n")
		oidsExist = readTestOids(args[0])
		oidsMissing = readTestOids(args[1])
	} else {
		fmt.Printf("Creating test data (will modify server contents)\n")
		oidsExist, oidsMissing = constructTestOids()
		// Run a 'test' which is really just a setup task, but because it has to
		// use the same APIs it's a test in its own right too
		err := runTest(ServerTest{"Set up test data", setupTestData}, endp, oidsExist, oidsMissing)
		if err != nil {
			exit("Failed to set up test data, aborting")
		}
	}

	runTests(endp, oidsExist, oidsMissing)
}

func readTestOids(filename string) []string {
	f, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		exit("Error opening file %s", filename)
	}
	defer f.Close()

	var ret []string
	rdr := bufio.NewReader(f)
	line, err := rdr.ReadString('\n')
	for err == nil {
		ret = append(ret, strings.TrimSpace(line))
		line, err = rdr.ReadString('\n')
	}

	return ret
}

func constructTestOids() (oidsExist, oidsMissing []string) {
	const oidCount = 50
	oidsExist = make([]string, 0, oidCount)
	oidsMissing = make([]string, 0, oidCount)

	// Generate SHAs, not random so repeatable
	rand.Seed(int64(oidCount))
	runningSha := sha256.New()
	for i := 0; i < oidCount; i++ {
		runningSha.Write([]byte{byte(rand.Intn(256))})
		oid := hex.EncodeToString(runningSha.Sum(nil))
		oidsExist = append(oidsExist, oid)

		runningSha.Write([]byte{byte(rand.Intn(256))})
		oid = hex.EncodeToString(runningSha.Sum(nil))
		oidsMissing = append(oidsMissing, oid)
	}
	return
}

func runTests(endp lfs.Endpoint, oidsExist, oidsMissing []string) {

	fmt.Printf("Running %d tests...\n", len(tests))
	for _, t := range tests {
		runTest(t, endp, oidsExist, oidsMissing)
	}

}

func runTest(t ServerTest, endp lfs.Endpoint, oidsExist, oidsMissing []string) error {
	const linelen = 70
	line := t.Name
	if len(line) > linelen {
		line = line[:linelen]
	} else if len(line) < linelen {
		line = fmt.Sprintf("%s%s", line, strings.Repeat(" ", linelen-len(line)))
	}
	fmt.Printf("%s...\r", line)

	err := t.F(endp, oidsExist, oidsMissing)
	if err != nil {
		fmt.Printf("%s FAILED\n", line)
		fmt.Println(err.Error())
	} else {
		fmt.Printf("%s OK\n", line)
	}
	return err
}

func setupTestData(endp lfs.Endpoint, oidsExist, oidsMissing []string) error {
	// TODO
	return nil
}

// Exit prints a formatted message and exits.
func exit(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(2)
}

func init() {
	RootCmd.Flags().StringVarP(&apiUrl, "url", "u", "", "URL of the API (must supply this or --clone)")
	RootCmd.Flags().StringVarP(&cloneUrl, "clone", "c", "", "Clone URL from which to find API (must supply this or --url)")
}
