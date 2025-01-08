package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

func errPanic(err error) {
	if nil != err {
		panic(err)
	}
}

type configer struct {
	API       string
	AppKey    string
	ScrectKey string
}

func main() {
	speaker := flag.Bool("s", false, "result speaker")
	flag.Parse()
	args := flag.Args()
	q := strings.Join(args, " ")
	if q == "-" {
		stdinBytes, err := ioutil.ReadAll(os.Stdin)
		errPanic(err)
		q = string(stdinBytes)
	}
	homeDir, err := os.UserHomeDir()
	errPanic(err)
	configBytes, err := ioutil.ReadFile(homeDir + "/.config/yd.json")
	errPanic(err)
	config := configer{}
	errPanic(json.Unmarshal(configBytes, &config))

	curtime := fmt.Sprintf("%d", time.Now().UTC().Unix())
	input := getInput(q)
	salt := time.Now().String()
	sign := getSign(config.AppKey, input, salt, curtime, config.ScrectKey)

	querys := url.Values{
		"q":        []string{q},
		"from":     []string{"auto"},
		"to":       []string{"auto"},
		"appKey":   []string{config.AppKey},
		"salt":     []string{salt},
		"sign":     []string{sign},
		"signType": []string{"v3"},
		"curtime":  []string{curtime},
		"ext":      []string{},
		"voice":    []string{},
		"strict":   []string{},
	}
	addr, err := url.Parse(config.API)
	errPanic(err)
	addr.RawQuery = querys.Encode()
	// fmt.Println("request", addr.String())
	resp, err := http.Get(addr.String())
	errPanic(err)
	respByte, err := ioutil.ReadAll(resp.Body)
	errPanic(err)
	// fmt.Println(string(respByte))
	respData := struct {
		Translation []string
		SpeakURL    string
		TSpeakURL   string
		Basic       struct {
			Phonetic string
			Explains []string
		}
		Web []struct {
			Key   string
			Value []string
		}
		ErrorCode string
	}{}
	errPanic(json.Unmarshal(respByte, &respData))
	if "0" != respData.ErrorCode {
		fmt.Println(string(respByte))
		return
	}
	fmt.Printf("%s[%s]: %s\n", q, respData.Basic.Phonetic, fmtArr(respData.Translation))
	fmt.Printf("\t%s\n", fmtArr(respData.Basic.Explains))
	fmt.Println("==================")
	for _, v := range respData.Web {
		fmt.Printf("%s: %s\n", v.Key, fmtArr(v.Value))
	}
	//fmt.Println(respData.TSpeakURL)
	if *speaker {
		cmd := exec.Command("/bin/mpg123", respData.TSpeakURL)
		_, err = cmd.Output()

		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	// fmt.Println(string(stdout))
}

func fmtArr(ss []string) string {
	return "\"" + strings.Join(ss, "\",\n\t\"") + "\""
}

func getInput(q string) string {
	if len(q) <= 20 {
		return q
	}
	return q[:10] + fmt.Sprintf("%d", len(q)) + q[len(q)-10:len(q)]
}

func getSign(appkey, input, salt, curtime, screctKey string) string {
	return fmt.Sprintf("%x",
		sha256.Sum256([]byte(appkey+input+salt+curtime+screctKey)))
}
