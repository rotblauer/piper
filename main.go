package main

import (
	"os"
	"bufio"
	"path/filepath"
	"flag"
	"io/ioutil"
	"strings"
	"strconv"
	"os/exec"
	"regexp"
	"errors"
	"fmt"
	"bytes"
)

var infilePath string
var outfilePath string
var shellCmd = "bash -c"
var baseCmdName = "cat"
var rawSeds []string
var errQuitting = errors.New("quitting")
var errContinue = errors.New("fake error: just continue")

func ensureAbsolutePath(s string) string {
	p, err := filepath.Abs(s)
	if err != nil {
		panic(err)
	}
	return p
}

func init() {
	flag.StringVar(&infilePath, "i", "./in.txt", "file to manipulate")
	flag.StringVar(&outfilePath, "o", "./out.txt", "file to generate")
	flag.StringVar(&baseCmdName, "b", "cat", "base command to grab from in file")
	flag.StringVar(&shellCmd, "s", "bash -c", "base command to grab from in file")
	flag.Parse()
}

func appendSed(s string) {
	rawSeds = append(rawSeds, s)
}

func editSed(i int, s []string) {
	var outSeds []string
	for j, val := range rawSeds {
		if j == i {
			outSeds = append(outSeds, strings.Join(s, " "))
		} else {
			outSeds = append(outSeds, val)
		}
	}
	rawSeds = outSeds
}

func insertSed(i int, s []string) {
	var out []string
	for j, v := range rawSeds {
		if j == i {
			out = append(out, strings.Join(s, " "))
		}
		out = append(out, v)
	}

	rawSeds = out
}

func rmSed(i int) {
	var outSeds []string
	for j, val := range rawSeds {
		if j != i {
			outSeds = append(outSeds, val)
		}
	}
	rawSeds = outSeds
}

func writeFile(p string, b []byte) {
	if err := ioutil.WriteFile(p, b, os.ModePerm); err != nil {
		panic(err)
	}
}

func concatSeds() string {
	return strings.Join(rawSeds, " ")
}

func sedsDisplayStringPretty() string {
	var os []string
	for i, v := range rawSeds {
		s := fmt.Sprintf("    [%d]  %s\n", i, v)
		os = append(os, s)
	}
	return strings.Join(os, "\n")
}

func save(p string) {
	pa, err := filepath.Abs(p)
	if err != nil {
		panic(err)
	}
	var data []byte
	for _, v := range rawSeds {
		vv := v + " \\\n"
		data = append(data, []byte(vv)...)
	}

	dir := filepath.Dir(pa)
	os.MkdirAll(dir, os.ModeDir)


	if err := ioutil.WriteFile(pa, data, os.ModePerm); err != nil {
		fmt.Println("Error saving cmd to:", pa, err)
	} else {
		fmt.Println("Saved cmd to:", pa)
	}
}

func load(p string) {
	pa, err := filepath.Abs(p)
	if err != nil {
		panic(err)
	}
	bs, err := ioutil.ReadFile(pa)
	if err != nil {
		fmt.Println("Could not read file:", pa, err)
		return
	}
	split := bytes.Split(bs, []byte(" \\\n"))
	rawSeds = []string{}
	for _, s := range split {
		v := bytes.TrimSuffix(s, []byte(" \\\n"))
		if string(v) != "" {
			rawSeds = append(rawSeds, string(v))
		}
	}
	fmt.Println("Loaded cmd from:", pa)
	fmt.Println(sedsDisplayStringPretty())
}

func printStatus(doneCommand string) {
	fmt.Println("--------------------------")
	fmt.Println("Executed:", doneCommand)
	fmt.Println("--------------------------")
	fmt.Println("Command:\n")
	fmt.Println(sedsDisplayStringPretty())
	fmt.Println(`
Awaiting command...`)
}

func handleInput(s string) (error) {
	quitRe := regexp.MustCompile(`^:q`)
	if quitRe.MatchString(s) {
		return errQuitting
	}

	// meta edit/rm controls
	if strings.HasPrefix(s, ":") {
		s = strings.TrimPrefix(s, ":")
		ss := strings.Split(s, " ")
		if len(ss) < 2 {
			if ss[0] == "h" {

				printUsage()
				fmt.Println("Awaiting command...")
				return errContinue
			} else {
				panic("use... :r 1 , :e 2 s/old/new/g ")
			}
		}
		i, err := strconv.Atoi(ss[1])
		if err != nil {
			switch ss[0] {
			case "w":
				save(ss[1])
				return errContinue
			case "l":
				load(ss[1])
				str, bs, err := executeCmd()
				if err != nil {
					fmt.Println("Error executing command:", err)
				}
				writeFile(outfilePath, bs)
				printStatus(str)

				return errContinue
			default:
				fmt.Println(ss[0], " = UNKNOWN COMMAND")
				printUsage()
				fmt.Println("Awaiting command...")
			}
			if ss[0] == "r" || ss[0] == "i" || ss[0] == "e" {
				return err
			}
		}
		switch ss[0] {
		case "r":
			rmSed(i)
		case "e":
			editSed(i, ss[2:])
		case "i":
			insertSed(i, ss[2:])
		default:
			fmt.Println(ss[0], " = UNKNOWN COMMAND")
			printUsage()
			fmt.Println("Awaiting command...")
		}
	} else {
		appendSed(s)
	}
	return nil
}

func executeCmd() (string, []byte, error) {
	var err error
	var bs []byte

	sc := shellCmd
	scs := strings.Split(sc, " ")

	var line string // is unified, legible string for printing
	var lines = []string{scs[0]} // always len 3

	if len(scs) > 1 {
		lines = append(lines, scs[1:]...)
	} else {
		lines = append(lines, "")
	}
	lines = append(lines, concatSeds())
	line = strings.Join(lines, " ")

	bs, err = exec.Command(lines[0], lines[1], lines[2]).Output()
	return line, bs, err
}

func printUsage() {
	fmt.Println(`
    | grep ok        <- append command '| grep ok'
    :r N             <- remove N command
    :e N | grep ok   <- change N command to '| grep ok'
    :i N | grep ok   <- insert command at index N as '| grep ok'
    :w ./file.sh     <- save runnable and loadable file from current command
    :l ./file.sh     <- load from file
    :h               <- help
    :q               <- quit
`)
}

func main() {
	infilePath = ensureAbsolutePath(infilePath)
	outfilePath = ensureAbsolutePath(outfilePath)

	// Print usage on config on startup
	fmt.Println("Reading in from:", infilePath)
	fmt.Println("Sending out to:", outfilePath)

	appendSed(baseCmdName + " " + infilePath)

	scanner := bufio.NewScanner(os.Stdin)
	if _, err := os.Create(outfilePath); err != nil {
		panic(err)
	}

	printUsage()

	str, bs, err := executeCmd()
	if err != nil {
		fmt.Println("Error executing command:", err)
	}
	writeFile(outfilePath, bs)
	printStatus(str)

	for scanner.Scan() {
		input := scanner.Text()
		err := handleInput(input)
		if err == errQuitting {
			break
		} else if err == errContinue {
			continue
		} else if err != nil {
			fmt.Println("abc")
			panic(err)
		}
		str, bs, err := executeCmd()
		if err != nil {
			fmt.Println("Error executing command:", err)
		}
		writeFile(outfilePath, bs)
		printStatus(str)
	}

	fmt.Println("Final command was:")
	fmt.Println(concatSeds())

	if err := scanner.Err(); err != nil {
		fmt.Println("jkl")
		fmt.Println("scanner error")
		panic(err)
	}
}
