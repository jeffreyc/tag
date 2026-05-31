package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"text/template"

	"github.com/fatih/color"
)

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func extractCmdExitCode(err error) int {
	if err != nil {
		// Extract real exit code
		// Source: https://stackoverflow.com/a/10385867
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}
		return 1
	}
	return 0
}

func optionIndex(args []string, option string) int {
	for i := len(args) - 1; i >= 0; i-- {
		if args[i] == option {
			return i
		}
	}
	return -1
}

func isatty(f *os.File) bool {
	stat, err := f.Stat()
	check(err)
	return stat.Mode()&os.ModeCharDevice != 0
}

func getEnvDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

var (
	red          = color.RedString
	blue         = color.BlueString
	lineNumberRe = regexp.MustCompile(`^(\d+):(\d+):.*`)
	ansi         = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`) // Source: https://superuser.com/a/380778
)

type AliasFile struct {
	filename  string
	aliasPref string
	cmdFmtStr string
	buf       bytes.Buffer
	writer    *bufio.Writer
}

func NewAliasFile() *AliasFile {
	aliasFilename := getEnvDefault("TAG_ALIAS_FILE", "/tmp/tag_aliases")
	aliasPrefix := getEnvDefault("TAG_ALIAS_PREFIX", "e")
	aliasCmdFmtString := getEnvDefault(
		"TAG_CMD_FMT_STRING",
		`vim -c "call cursor({{.LineNumber}}, {{.ColumnNumber}})" "{{.Filename}}"`)

	a := &AliasFile{
		aliasPref: aliasPrefix,
		cmdFmtStr: aliasCmdFmtString,
		filename:  aliasFilename,
	}
	a.writer = bufio.NewWriter(&a.buf)
	a.WriteUnaliases(readAliasNames(aliasFilename, aliasPrefix))
	return a
}

func readAliasNames(filename, prefix string) []string {
	data, err := ioutil.ReadFile(filename)
	if os.IsNotExist(err) {
		return nil
	}
	check(err)

	var names []string
	for _, line := range strings.Split(string(data), "\n") {
		aliasDef, ok := strings.CutPrefix(line, "alias ")
		if !ok {
			continue
		}
		name, _, ok := strings.Cut(aliasDef, "=")
		if ok && strings.HasPrefix(name, prefix) && allDigits(strings.TrimPrefix(name, prefix)) {
			names = append(names, name)
		}
	}
	return names
}

func allDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func shellDoubleQuoteEscape(s string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"$", "\\$",
		"`", "\\`",
	)
	return replacer.Replace(s)
}

func (a *AliasFile) WriteAlias(index int, filename, linenum string, colnum string) {
	t := template.Must(template.New("alias").Parse(a.cmdFmtStr))

	aliasVars := struct {
		MatchIndex   int
		Filename     string
		LineNumber   string
		ColumnNumber string
	}{index, shellDoubleQuoteEscape(filename), linenum, colnum}

	var cmd bytes.Buffer
	err := t.Execute(&cmd, aliasVars)
	check(err)

	_, err = fmt.Fprintf(a.writer, "alias %s%s=%s\n", a.aliasPref, fmt.Sprint(index), shellSingleQuote(cmd.String()))
	check(err)
}

func (a *AliasFile) WriteUnaliases(names []string) {
	for _, name := range names {
		_, err := fmt.Fprintf(a.writer, "unalias %s 2>/dev/null || true\n", shellSingleQuote(name))
		check(err)
	}
}

func (a *AliasFile) WriteFile() {
	err := a.writer.Flush()
	check(err)

	err = ioutil.WriteFile(a.filename, a.buf.Bytes(), 0644)
	check(err)
}

func tagPrefix(aliasIndex int) string {
	return blue("[") + red("%d", aliasIndex) + blue("]")
}

func generateTags(cmd *exec.Cmd) int {
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	check(err)

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	scanner.Split(bufio.ScanLines)

	var (
		line          []byte
		colorlessLine []byte
		curPath       string
		groupIdxs     []int
	)

	aliasFile := NewAliasFile()
	defer aliasFile.WriteFile()

	aliasIndex := 1

	err = cmd.Start()
	check(err)

	for scanner.Scan() {
		line = scanner.Bytes()
		colorlessLine = ansi.ReplaceAll(line, nil) // strip ANSI
		if len(curPath) == 0 {
			// Path is always in the first line of a group (the heading). Extract and print it
			curPath = string(colorlessLine)
			curPath, err = filepath.Abs(curPath)
			check(err)
			fmt.Println(string(line))
		} else if groupIdxs = lineNumberRe.FindSubmatchIndex(colorlessLine); len(groupIdxs) > 0 {
			// Extract and tag matches
			aliasFile.WriteAlias(
				aliasIndex,
				curPath,
				string(colorlessLine[groupIdxs[2]:groupIdxs[3]]),
				string(colorlessLine[groupIdxs[4]:groupIdxs[5]]))
			fmt.Printf("%s %s\n", tagPrefix(aliasIndex), string(line))
			aliasIndex++
		} else {
			// Empty line. End of grouping, reset curPath context
			fmt.Println(string(line))
			curPath = ""
		}
	}
	check(scanner.Err())

	err = cmd.Wait()
	return extractCmdExitCode(err)
}

func passThrough(cmd *exec.Cmd) int {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	return extractCmdExitCode(err)
}

func validateSearchProg(prog string) error {
	switch prog {
	case "ag", "rg":
		return nil
	default:
		return fmt.Errorf(
			"invalid environment variable TAG_SEARCH_PROG='%s'. only 'ag' and 'rg' are supported.",
			prog)
	}
}

func constructTagArgs(searchProg string, userArgs []string) []string {
	if isatty(os.Stdout) {
		switch searchProg {
		case "ag":
			return []string{"--group", "--color", "--column"}
		case "rg":
			// ripgrep can't handle more than one --color option, so if the user provides one
			// we have to explicilty keep tag from passing its own --color option
			if _, ok := colorOptionValue(userArgs); ok {
				return []string{"--heading", "--with-filename", "--column"}
			}
			return []string{"--heading", "--with-filename", "--color", "always", "--column"}
		}
	}
	return []string{}
}

func colorOptionValue(args []string) (string, bool) {
	for i := len(args) - 1; i >= 0; i-- {
		if args[i] == "--color" {
			if i+1 < len(args) {
				return args[i+1], true
			}
			return "", true
		}
		if value, ok := strings.CutPrefix(args[i], "--color="); ok {
			return value, true
		}
	}
	return "", false
}

func handleColorSetting(prog string, args []string) {
	switch prog {
	case "ag":
		color.NoColor = (optionIndex(args, "--nocolor") >= 0)
	case "rg":
		colorValue, _ := colorOptionValue(args)
		color.NoColor = (colorValue == "never")
	}
}

func main() {
	searchProg := getEnvDefault("TAG_SEARCH_PROG", "ag")
	check(validateSearchProg(searchProg))

	userArgs := os.Args[1:]

	disableTag := false
	var tagArgs []string

	switch i := optionIndex(userArgs, "--notag"); {
	case i >= 0:
		userArgs = append(userArgs[:i], userArgs[i+1:]...)
		fallthrough
	case len(userArgs) == 0: // no arguments; fall back to help message
		disableTag = true
	default:
		tagArgs = constructTagArgs(searchProg, userArgs)
	}
	finalArgs := append(tagArgs, userArgs...)

	cmd := exec.Command(searchProg, finalArgs...)

	if disableTag || !isatty(os.Stdin) || !isatty(os.Stdout) {
		// Data being piped from stdin
		os.Exit(passThrough(cmd))
	}

	handleColorSetting(searchProg, finalArgs)
	os.Exit(generateTags(cmd))
}
