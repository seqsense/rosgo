package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func writeCode(fullname string, code string) error {
	nameComponents := strings.Split(fullname, "/")
	pkgDir := filepath.Join(dirName, nameComponents[0])
	if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
		err = os.MkdirAll(pkgDir, os.ModeDir|os.FileMode(0775))
		if err != nil {
			return err
		}
	}
	filename := filepath.Join(pkgDir, nameComponents[1]+".go")

	return ioutil.WriteFile(filename, []byte(code), os.FileMode(0664))
}

var (
	dirName    = "vendor"
	flagVendor = flag.Bool("vendor", true, "Let vendoring or not")
)

func main() {
	flag.Parse()
	if len(flag.Args()) < 2 {
		fmt.Println("USAGE: gengo msg|srv <NAME> [<FILE>]")
		os.Exit(-1)
	}

	mode := flag.Args()[0]
	if !(*flagVendor) {
		switch mode {
		case "msg":
			dirName = "rosmsgs"
		case "srv":
			dirName = "rossrvs"
		}
	}
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		err = os.Mkdir(dirName, os.ModeDir|os.FileMode(0775))
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
	}

	rosPkgPath := os.Getenv("ROS_PACKAGE_PATH")
	context, err := NewMsgContext(strings.Split(rosPkgPath, ":"))
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	fullname := flag.Args()[1]
	fmt.Printf("Generating %v...", fullname)

	if mode == "msg" {
		var spec *MsgSpec
		var err error
		if len(flag.Args()) == 2 {
			spec, err = context.LoadMsg(fullname)
		} else {
			spec, err = context.LoadMsgFromFile(flag.Args()[2], fullname)
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		var code string
		code, err = GenerateMessage(context, spec)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		err = writeCode(fullname, code)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
	} else if mode == "srv" {
		var spec *SrvSpec
		var err error
		if len(flag.Args()) == 2 {
			spec, err = context.LoadSrv(fullname)
		} else {
			spec, err = context.LoadSrvFromFile(flag.Args()[2], fullname)
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		srvCode, reqCode, resCode, err := GenerateService(context, spec)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}

		err = writeCode(fullname, srvCode)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}

		err = writeCode(spec.Request.FullName, reqCode)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}

		err = writeCode(spec.Response.FullName, resCode)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
	} else {
		fmt.Println("USAGE: genmsg <MSG>")
		os.Exit(-1)
	}
	fmt.Println("Done")
}
