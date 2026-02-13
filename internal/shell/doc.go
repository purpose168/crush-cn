package shell

// shell 包使用示例:
//
// 1. 执行一次性命令:
//
//	shell := shell.NewShell(nil)
//	stdout, stderr, err := shell.Exec(context.Background(), "echo hello")
//
// 2. 在多个命令之间维护状态:
//
//	shell := shell.NewShell(&shell.Options{
//	    WorkingDir: "/tmp",
//	    Logger: myLogger,
//	})
//	shell.Exec(ctx, "export FOO=bar")
//	shell.Exec(ctx, "echo $FOO")  // 将输出 "bar"
//
// 3. 管理环境变量和工作目录:
//
//	shell := shell.NewShell(nil)
//	shell.SetEnv("MY_VAR", "value")
//	shell.SetWorkingDir("/tmp")
//	cwd := shell.GetWorkingDir()
//	env := shell.GetEnv()
