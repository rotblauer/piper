package main

import (
	"fmt"
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
	"log"
	"io"
	"bytes"
)

var infilePath string
var outfilePath string
var baseCmdName = "cat"
var rawSeds []string
var seds []*exec.Cmd // hold a set of arbitrary operations
var errQuitting = errors.New("quitting")

func ensureAbsolutePath(s string) string {
	p, err := filepath.Abs(s)
	if err != nil {
		panic(err)
	}
	return p
}

func parseRawSedsToCmds() []*exec.Cmd {
	var o []*exec.Cmd
	for _, v := range rawSeds {
		ss := strings.Split(v, " ")
		if len(ss) > 1 {
			o = append(o, exec.Command(ss[0], ss[1:]...))
		} else {
			o = append(o, exec.Command(ss[0]))
		}
	}
	return o
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

func editSed(i int, s string) {
	var outSeds []string
	for j, val := range rawSeds {
		if j == i {
			outSeds = append(outSeds, s)
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
	var os []string
	for _, v := range seds {
		s := strings.Join(v.Args, " ")
		os = append(os, s)
	}
	return strings.Join(os, " | ")
}

func buildCmds() []*exec.Cmd {
	var cmds []*exec.Cmd
	cmds = append(cmds, exec.Command(baseCmdName, infilePath))
	cmds = append(cmds, parseRawSedsToCmds()...)
	return cmds
}

func handleInput(s string) ([]*exec.Cmd, error) {
	quitRe := regexp.MustCompile(`(quit|exit)`)
	if quitRe.MatchString(s) {
		return nil, errQuitting
	}

	// meta edit/rm controls
	if strings.HasPrefix(s, ":") {
		s = strings.TrimPrefix(s, ":")
		ss := strings.Split(s, " ")
		if len(ss) >= 2 {
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
			editSed(i, ss[2])
		default:
		}
	} else {
		addSed(s)
	}
	return buildCmds(), nil
}

func main() {
	infilePath = ensureAbsolutePath(infilePath)

	scanner := bufio.NewScanner(os.Stdin)
	if _, err := os.Create(outfilePath); err != nil {
		panic(err)
	}
	for scanner.Scan() {
		var b bytes.Buffer
		input := scanner.Text()
		cmds, err := handleInput(input)
		if err == errQuitting {
			break
		} else if err != nil {
			panic(err)
		} else if cmds == nil {
			panic("cmds are nil")
		}
		seds = buildCmds()

		log.Println("seds",seds)
		log.Println("rawSeds", rawSeds)
		fmt.Println(sedsDisplayString())

		log.Println("doing")
		if err := Execute(&b,
			exec.Command("grep", "-E", "'disc'")); err != nil {
			panic(err)
		}
		writeFile(outfilePath, b.Bytes())

	}

	if err := scanner.Err(); err != nil {
		log.Println("scanner error")
		panic(err)
	}
}

func Execute(output_buffer *bytes.Buffer, stack ...*exec.Cmd) (err error) {
	var error_buffer bytes.Buffer
	pipe_stack := make([]*io.PipeWriter, len(stack)-1)
	i := 0
	for ; i < len(stack)-1; i++ {
		stdin_pipe, stdout_pipe := io.Pipe()
		stack[i].Stdout = stdout_pipe
		stack[i].Stderr = &error_buffer
		stack[i+1].Stdin = stdin_pipe
		pipe_stack[i] = stdout_pipe
	}
	stack[i].Stdout = output_buffer
	stack[i].Stderr = &error_buffer

	if err := call(stack, pipe_stack); err != nil {
		log.Println("got err")
		log.Fatalln(string(error_buffer.Bytes()), err)
	}
	return err
}

func call(stack []*exec.Cmd, pipes []*io.PipeWriter) (err error) {
	if stack[0].Process == nil {
		log.Println("starting 0")
		if err = stack[0].Start(); err != nil {
			log.Println("goterrrr", err)
			log.Println(stack[0].Args)
			return err
		} else {
			log.Println("was ok", stack[0].Args)
		}
	}
	if len(stack) > 1 {
		log.Println("+1")
		if err = stack[1].Start(); err != nil {
			log.Println("er1+", err)
			return err
		}
		defer func() {
			if err == nil {
				pipes[0].Close()
				err = call(stack[1:], pipes[1:])
				if err != nil {
					log.Println("defererr+", err)
				}
			}
		}()
	}
	e := stack[0].Wait()
	if e != nil {
		log.Println("waiterr", e)
	}
	return e
}