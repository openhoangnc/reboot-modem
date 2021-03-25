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
	log.Println("-> login:")
	loginActionURL := *modemURL + "/cgi-bin/login_action.cgi"
	loginPayload := strings.NewReader(fmt.Sprintf(`action=login&txtUserId=%s&txtPassword=%s`, *userID, *password))
	log.Println("  -> post:", loginActionURL)
	loginResponse, err := client.Post(loginActionURL, "application/x-www-form-urlencoded", loginPayload)
	if err != nil {
		log.Println("    -> error:", err)
		os.Exit(1)
	}
	log.Println("  -> post: done")
	loginResponse.Body.Close()

	log.Println("-> login: done")
}

// Get DSToken and save to global variables
func getDSToken() {
	log.Println("-> getDSToken:")

	rebootURL := *modemURL + "/cgi-bin/reboot.cgi"

	log.Println("  -> get:", rebootURL)
	rebootResponse, err := client.Get(rebootURL)
	if err != nil {
		log.Println("    -> error:", err)
		os.Exit(1)
	}
	log.Println("  -> get: done")
	rebootResponse.Body.Close()

	log.Println("  -> read response:")
	rebootPage, err := ioutil.ReadAll(rebootResponse.Body)
	if err != nil {
		log.Println("    -> error:", err)
		os.Exit(1)
	}
	log.Println("  -> read response: done")

	log.Println("  -> find DSToken:")
	matches := regexp.MustCompile("name='DSToken' value='([^']+)'").FindStringSubmatch(string(rebootPage))
	if len(matches) != 2 {
		log.Println("    -> not found:")
		log.Println(string(rebootPage))
		os.Exit(1)
	}
	log.Println("  -> find DSToken: done")
	dsToken = matches[1]

	log.Println("-> getDSToken: done")
}

func reboot() {
	log.Println("-> reboot:")

	rebootActionURL := *modemURL + "/cgi-bin/reboot_action.cgi"
	rebootActionPayload := strings.NewReader(fmt.Sprintf(`waiting_action=1&DSToken=%s`, dsToken))

	log.Println("  -> post:", rebootActionURL)
	response, err := client.Post(rebootActionURL, "application/x-www-form-urlencoded", rebootActionPayload)
	if err != nil {
		log.Println("    -> error:", err)
		os.Exit(1)
	}
	log.Println("  -> post: done")

	response.Body.Close()

	log.Println("-> reboot: done")
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
	log.SetOutput(os.Stdout)

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
			waitDuration := -time.Until(waitBegin)
			if waitDuration.Seconds() > 1 {
				fmt.Println()
			}
			log.Println("New PublicIP:", publicIP)
			log.Println("Rebooting duration:", waitDuration.String())
			return
		}
		fmt.Print(".")
		time.Sleep(1 * time.Second)
	}
}
