package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Bash struct {
	Dir    string
	Silent bool
}

func (r *Bash) Run(args ...string) (err error) {
	command := strings.Join(args, " ")
	if !r.Silent {
		fmt.Printf("[CMD] %s\n", command)
	}
	cmd := exec.Command("bash", "--norc", "-c", command)
	cmd.Dir = r.Dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return
	}
	if !r.Silent {
		fmt.Print(string(output))
	}
	return
}

func (r *Bash) Ask(prompt string, v ...any) (b bool) {
	if YesAssumed {
		b = true
		return
	}
	fmt.Printf(prompt, v...)
	for {
		fmt.Print(" [Y|n]: ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if answer != "" {
			switch answer[0] {
			case '\n', 'Y', 'y':
				b = true
				return
			case 'N', 'n':
				return
			}
		}
	}
}
