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
	"log"
)

var infilePath string
var outfilePath string
var baseCmdName = "cat"
var rawSeds []string
var errQuitting = errors.New("quitting")

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
}

//func readFile(p string) []byte {
//	b, err := ioutil.ReadFile(p)
//	if err != nil {
//		panic(err)
//	}
//	return b
//}

func addSed(s string) {
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

func sedsDisplayString() string {
	return strings.Join(rawSeds, " | ")
}

func sedsDisplayStringPretty() string {
	var os []string
	for i, v := range rawSeds {
		s := fmt.Sprintf("%d[%s]\n", i, v)
		os = append(os, s)
	}
	return strings.Join(os, "\n")
}

func handleInput(s string) (error) {
	quitRe := regexp.MustCompile(`(quit|exit)`)
	if quitRe.MatchString(s) {
		return errQuitting
	}

	// meta edit/rm controls
	if strings.HasPrefix(s, ":") {
		s = strings.TrimPrefix(s, ":")
		ss := strings.Split(s, " ")
		if len(ss) < 2 {
			panic("use... :rm 1 , :e 2 s/old/new/g ")
		}
		i, err := strconv.Atoi(ss[1])
		if err != nil {
			panic(err)
		}
		switch ss[0] {
		case "rm":
			rmSed(i)
		case "e":
			editSed(i, ss[2:])
		default:
		}
	} else {
		addSed(s)
	}
	return nil
}

func main() {
	infilePath = ensureAbsolutePath(infilePath)

	addSed(baseCmdName + " " + infilePath)

	scanner := bufio.NewScanner(os.Stdin)
	if _, err := os.Create(outfilePath); err != nil {
		panic(err)
	}
	fmt.Println("Sending out to:", ensureAbsolutePath(outfilePath))
	fmt.Println(" ")
	fmt.Println("    :rm N", "<- remove N command")
	fmt.Println("    :e 1 grep ok", "<- change N command to 'grep ok'")
	fmt.Println(" ")
	fmt.Println("Enter your chainable filter command:")
	for scanner.Scan() {
		input := scanner.Text()
		err := handleInput(input)
		if err == errQuitting {
			break
		} else if err != nil {
			fmt.Println("abc")
			panic(err)
		}

		fmt.Println("---------")
		fmt.Println("Command:\n")
		fmt.Println(sedsDisplayStringPretty())
		fmt.Println("  -> ", sedsDisplayString(), "\n")


		bs, err := exec.Command("bash", "-c", sedsDisplayString()).Output()
		if err != nil {
			log.Println("Error executing command.")
			//	rmSed(len(rawSeds)-1) // remove last one
			//
			//	fmt.Println("---------")
			//	fmt.Println("Command:\n")
			//	fmt.Println(sedsDisplayStringPretty())
			//	bs = []byte("")
			//}
		}
		writeFile(outfilePath, bs)
		fmt.Println("\nEnter your chainable filter command:")
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("jkl")
		fmt.Println("scanner error")
		panic(err)
	}
}
