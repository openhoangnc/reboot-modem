package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	modemURL = flag.String("url", "https://192.168.1.100:8080", "URL")
	userID   = flag.String("user", "admin", "User")
	password = flag.String("pwd", "", "Password")
	client   *http.Client
	dsToken  string
)

func initClient() {
	client = &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	client.Jar, _ = cookiejar.New(nil)
}

func checkFlags() {
	flag.Parse()
	if len(*password) == 0 {
		flag.Usage()
		os.Exit(1)
	}
}

// Login will set some cookies to client after login successful
func login() {
	loginActionURL := *modemURL + "/cgi-bin/login_action.cgi"
	loginPayload := strings.NewReader(fmt.Sprintf(`action=login&txtUserId=%s&txtPassword=%s`, *userID, *password))
	loginResponse, err := client.Post(loginActionURL, "application/x-www-form-urlencoded", loginPayload)
	if err != nil {
		log.Println("login post login_action.cgi", err)
		os.Exit(1)
	}
	loginResponse.Body.Close()
}

// Get DSToken and save to global variables
func getDSToken() {
	rebootURL := *modemURL + "/cgi-bin/reboot.cgi"
	rebootResponse, err := client.Get(rebootURL)
	if err != nil {
		log.Println("getDSToken get reboot.cgi", err)
		os.Exit(1)
	}
	defer rebootResponse.Body.Close()

	rebootPage, err := ioutil.ReadAll(rebootResponse.Body)
	if err != nil {
		log.Println("getDSToken read response body", err)
		os.Exit(1)
	}

	matches := regexp.MustCompile("name='DSToken' value='([^']+)'").FindStringSubmatch(string(rebootPage))
	if len(matches) != 2 {
		log.Println("getDSToken not found")
		log.Println(string(rebootPage))
		os.Exit(1)
	}
	dsToken = matches[1]
}

func reboot() {
	rebootActionURL := *modemURL + "/cgi-bin/reboot_action.cgi"
	rebootActionPayload := strings.NewReader(fmt.Sprintf(`waiting_action=1&DSToken=%s`, dsToken))
	response, err := client.Post(rebootActionURL, "application/x-www-form-urlencoded", rebootActionPayload)
	if err != nil {
		log.Println("reboot post reboot_action.cgi", err)
		os.Exit(1)
	}
	response.Body.Close()
}

func getPublicIP() string {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://ip1.dynupdate.no-ip.com/")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(b)
}

func main() {
	initClient()
	checkFlags()

	log.Println("Reboot-modem")
	log.Println("Current PublicIP:", getPublicIP())

	login()
	getDSToken()
	reboot()

	log.Println("Reboot requested")
	log.Println("Checking for new public IP")
	waitBegin := time.Now()
	for {
		publicIP := getPublicIP()
		if len(publicIP) > 0 {
			waitDuration := -time.Until(waitBegin).Seconds()
			if waitDuration > 1 {
				fmt.Println()
			}
			log.Println("New PublicIP:", publicIP)
			return
		}
		fmt.Print(".")
		time.Sleep(1 * time.Second)
	}
}
