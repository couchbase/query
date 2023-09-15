package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/couchbase/query/errors"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("expected usage->\n ./finderr [code] : get error related information\n" +
			" ./finderr -s [pattern] : get all errornames that match the pattern")
	}

	// flag passed
	if os.Args[1][0] == '-' {
		flag := os.Args[1]
		if flag == "--search" || flag == "-s" {
			if len(os.Args) < 3 {
				log.Fatal("--search flag: expects pattern as an arguemnt: -s [pattern]")
			}

			errs, ok := errors.SearchError(strings.ToUpper(os.Args[2]), true)
			if !ok {
				fmt.Println("No matching errors!")
				return
			}

			fmt.Printf("Name-\t\tCode\n")
			for errname, errcode := range errs {
				fmt.Printf("%v-\t%v\n", errname, errcode)
			}
		} else {
			log.Fatal("invalid flag:\n supported flags:\n -s, --search [pattern]: lookup errornames similar to the pattern recieved")
		}

		return
	}

	// no flags passed
	codestring := os.Args[1]
	if codestring[0] == '-' {
		log.Fatal("expected 2 arguments when using flags-> [flag] [flag- arg]")
	}
	code, cerr := strconv.Atoi(codestring)
	if cerr != nil {
		codes, ok := errors.SearchError(codestring, false)
		if !ok {
			log.Fatal("expected usage-> [code]-> doesn't represent a defined errorcode name")
		}

		for _, val := range codes {
			code = val
			break
		}
	}

	errData, ok := errors.DescribeError(errors.ErrorCode(code))
	if !ok {
		fmt.Println("Manual not updated for this error code [", code, "]:(")
		return
	}

	tempFile, err := os.CreateTemp("", "man")
	if err != nil {
		log.Fatal("Error creating temporary file:", err)
	}
	defer os.Remove(tempFile.Name())
	// temp file created

	// create man page
	tempFile.Write([]byte(`.TH man 8 "11 01 2023" "1.0" "finderr"`))
	tempFile.Write([]byte("\n"))

	tempFile.Write([]byte(".SH CODE\n"))
	tempFile.Write([]byte(fmt.Sprintf("%v\n", errData.Code)))

	tempFile.Write([]byte(".SH NAME\n"))
	tempFile.Write([]byte(fmt.Sprintf("%v\n", errData.ErrorCode)))

	tempFile.Write([]byte(".SH DESCRIPTION\n"))
	tempFile.Write([]byte(fmt.Sprintf("%v\n", errData.Description)))

	var s1 string
	tempFile.Write([]byte(".SH CAUSE\n"))
	if len(errData.Causes) == 0 {
		tempFile.Write([]byte("None\n"))
	} else {
		for _, cause := range errData.Causes {
			s1 = strings.Replace(cause, "\\", "\\\\", -1)
			tempFile.Write([]byte(fmt.Sprintf("%v\n\n", s1)))
		}
		tempFile.Write([]byte("\n"))
	}

	tempFile.Write([]byte(".SH ACTIONS\n"))
	if len(errData.Actions) == 0 {
		tempFile.Write([]byte("None\n"))
	} else {
		for _, action := range errData.Actions {
			s1 = strings.Replace(action, "\\", "\\\\", -1)
			tempFile.Write([]byte(fmt.Sprintf("%v\n\n", s1)))
		}
		tempFile.Write([]byte("\n"))
	}

	tempFile.Write([]byte(".SH USER ERROR\n"))
	if errData.IsUser {
		tempFile.Write([]byte("Yes"))
	} else {
		tempFile.Write([]byte("No"))
	}

	cmd := exec.Command("man", tempFile.Name())
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
