//
// Created by Zhenwen Xu
// Modified by ChenYuZhao on 2020/7/3.
//
package api

import (
	"bufio"
	"fmt"
	"github.com/cihub/seelog"
	"github.com/gin-gonic/gin"
	"lmp/config"
	"lmp/daemon"
	"lmp/internal/BPF"
	"lmp/pkg/model"
	"net/http"
	"os/exec"
	"strings"
)

func init() {
	SetRouterRegister(func(router *RouterGroup) {
		engine := router.Group("/api")

		engine.GET("/ping", Ping)
		engine.POST("/data/collect", Do_collect)
		engine.POST("/register", UserRegister)
		engine.POST("/login", UserLogin)
		engine.POST("/uploadfiles", UpLoadFiles)
		engine.GET("/service", PrintService)
	})
}

func Ping(c *Context) {
	c.JSON(200, gin.H{"message": "pong"})
}

func Do_collect(c *Context) {
	//生成配置
	m := fillConfigMessage(c)
	//fmt.Println(m)
	//fmt.Println(m.BpfFilePath)

	for _,filePath := range m.BpfFilePath {
		//fmt.Println(filePath)
		go execute(filePath,m)
	}

	c.Redirect(http.StatusMovedPermanently, "http://"+config.GrafanaIp)
	return
}

func execute(filepath string, m model.ConfigMessage) {
	var newScript string
	// If pidflag is true, then we should add the pid parameter
	if m.PidFlag == true {
		script := make([]string, 0)
		script = append(script, "-P")
		script = append(script, m.Pid)
		newScript = strings.Join(script, " ")
	} else {
		newScript = ""
	}
	// fmt.Println(filepath)
	fmt.Println("[ConfigMessage] :", m.PidFlag, m.Pid)
	fmt.Println("[string] :", filepath, newScript)
	cmd := exec.Command("sudo", "python", filepath, newScript)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		seelog.Error(err)
		return
	}
	defer stdout.Close()
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
		}
	}()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		seelog.Error(err)
		return
	}
	defer stderr.Close()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
		}
	}()

	err = cmd.Start()
	if err != nil {
		seelog.Error(err)
		return
	}

	err = cmd.Wait()
	if err != nil {
		seelog.Error(err)
		return
	}
	seelog.Info("start extracting data...")
}

func fillConfigMessage(c *Context) model.ConfigMessage {
	var m model.ConfigMessage

	if _, ok := c.GetPostForm("dispatchingdelay"); ok {
		m.DispatchingDelay = true
		m.BpfFilePath = append(m.BpfFilePath, config.PluginPath + "dispatchingdelay.py")
	} else {
		m.DispatchingDelay = false
	}
	if _, ok := c.GetPostForm("waitingqueuelength"); ok {
		m.WaitingQueueLength = true
		m.BpfFilePath = append(m.BpfFilePath, config.PluginPath + "waitingqueuelength.py")
	} else {
		m.WaitingQueueLength = false
	}
	if _, ok := c.GetPostForm("softirqtime"); ok {
		m.SoftIrqTime = true
		m.BpfFilePath = append(m.BpfFilePath, config.PluginPath + "softirqtime.py")
	} else {
		m.SoftIrqTime = false
	}
	if _, ok := c.GetPostForm("hardirqtime"); ok {
		m.HardIrqTime = true
		m.BpfFilePath = append(m.BpfFilePath, config.PluginPath + "hardirqtime.py")
	} else {
		m.HardIrqTime = false
	}
	if _, ok := c.GetPostForm("oncputime"); ok {
		m.OnCpuTime = true
		m.BpfFilePath = append(m.BpfFilePath, config.PluginPath + "oncputime.py")
	} else {
		m.OnCpuTime = false
	}
	if _, ok := c.GetPostForm("vfsstat"); ok {
		m.Vfsstat = true
		m.BpfFilePath = append(m.BpfFilePath, config.PluginPath + "vfsstat.py")
	} else {
		m.Vfsstat = false
	}
	if _, ok := c.GetPostForm("dcache"); ok {
		m.Vfsstat = true
		m.BpfFilePath = append(m.BpfFilePath, config.PluginPath + "dcache.py")
	} else {
		m.Dcache = false
	}
	// Todo : Is this PID process exists?
	if _, ok := c.GetPostForm("pid"); ok {
		m.PidFlag = true
		// Then store the real pid number
		if pid, ok := c.GetPostForm("pidnum"); ok {
			if pid != "-1" {
				m.Pid = pid
			} else {
				m.PidFlag = false
			}
		} else {
			m.PidFlag = false
		}
	} else {
		m.PidFlag = false
	}

	return m
}

//用户注册处理器函数
func UserRegister(c *Context) {
	//接收前端传入的参数，并绑定到一个UserModel结构体变量中
	var user model.UserModel
	if err := c.ShouldBind(&user); err != nil {
		seelog.Error("err ->", err.Error())
		c.String(http.StatusBadRequest, "输入的数据不合法")
	}

	//接收数据合法后，存入数据库mysql
	/*
		passwordAgain := c.PostForm("password-again")
		if passwordAgain != user.Password {
			c.String(http.StatusBadRequest, "密码校验无效，两次密码不一致")
			log.Panicln("密码校验无效，两次密码不一致")
		}
	*/
	id := user.Save()
	seelog.Info("username", user.Username, "password", user.Password, "password again", user.PasswordAgain)

	seelog.Info("id is ", id)
	fmt.Println(id)
	fmt.Println(user)
	c.File(fmt.Sprintf("%s/login.html", "static"))
}

//用户登录处理器函数
func UserLogin(c *Context) {
	var user model.UserModel
	if e := c.Bind(&user); e != nil {
		seelog.Error("login 绑定错误", e.Error())
	}

	u := user.QueryByEmail()
	if u.Password == user.Password {
		seelog.Info("登录成功", u.Username)
		c.File(fmt.Sprintf("%s/index.html", "static"))
	}
}

// Delete later
func UpLoadFile(c *Context) {
	file, err := c.FormFile("bpffile")
	if err != nil {
		c.String(http.StatusBadRequest, "FormFile failed")
		return
	}
	fmt.Println("file:", file.Filename)

	if err := c.SaveUploadedFile(file, config.PluginPath); err != nil {
		c.String(http.StatusBadRequest, "upload failed:%s", err.Error())
		return
	}
	c.String(http.StatusOK, "upload success")
}

func UpLoadFiles(c *Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
	}
	// todo :  Multiple files uploaded
	files := form.File["bpffile"]
	fmt.Println("file:", files)
	fmt.Println(config.PluginPath)

	for _, file := range files {
		seelog.Info(file.Filename)
		c.SaveUploadedFile(file, config.PluginPath + file.Filename)
		// Put the name of the newly uploaded plug-in into the pipeline Filename
		daemon.FileChan <- file.Filename
	}

	c.String(http.StatusOK, fmt.Sprintf("%d files uploaded!", len(files)))
}

// Feedback existing plugins to the front end
func PrintService(c *Context) {
	var pluginsName []string

	for _, plugin := range bpf.PluginServices {
		pluginsName = append(pluginsName, plugin.Name)
	}

	c.JSON(http.StatusOK, gin.H{
		"plugins": pluginsName,
	})
}