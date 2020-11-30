package tools

import (
	"errors"
	"fmt"
	"ays/src/modules/graceful"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"golang.org/x/net/context"
)

type Result struct {
	output string
	err    error
}

// 执行shell命令，可设置执行超时时间
func ExecShell(ctx context.Context, command string) (string, error) {
	graceful.AddJob() // 记录执行中的任务，进程退出前等待任务执行
	defer graceful.FinishJob() // 标记任务执行完成

	cmd := exec.Command("/bin/bash", "-c", command)
	importEnv(cmd)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	resultChan := make(chan Result)
	go func() {
		output, err := cmd.CombinedOutput()
		fmt.Println("------------command-res--------------")
		fmt.Println(string(output))
		if err != nil {
			fmt.Println("+++++++++++got-err++++++")
			fmt.Println(err.Error())
		}
		fmt.Println("-------------------------------------")
		resultChan <- Result{string(output), err}
	}()
	select {
	case <-ctx.Done():
		if cmd.Process.Pid > 0 {
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return "", errors.New("timeout killed")
	case result := <-resultChan:
		return result.output, result.err
	}
}

func importEnv(cmd *exec.Cmd) {
	cmd.Env = os.Environ()
	dat, err := ioutil.ReadFile("/etc/environment")
	if err != nil {
		fmt.Println("read /etc/environment failed")
		fmt.Println(err.Error())
		return
	}
	etcEnv := strings.Split(string(dat), "\n")
	for _, val := range etcEnv {
		if val == "" {
			continue
		}
		cmd.Env = append(cmd.Env, val)
	}
}