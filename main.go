package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"regexp"
	"strings"
	"time"
)

const maxHttpRetry = 5

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
		Timeout:   3 * time.Second,
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

	for try := 0; try < maxHttpRetry; try++ {
		loginResponse, err := client.Post(loginActionURL, "application/x-www-form-urlencoded", loginPayload)
		if err != nil {
			if try < maxHttpRetry {
				continue
			}
			log.Println("    -> error:", err)
			os.Exit(1)
		}
		log.Println("  -> post: done")
		ioutil.ReadAll(loginResponse.Body)
		loginResponse.Body.Close()
		break
	}

	log.Println("-> login: done")
}

// Get DSToken and save to global variables
func getDSToken() {
	log.Println("-> getDSToken:")

	rebootURL := *modemURL + "/cgi-bin/reboot.cgi"

	log.Println("  -> get:", rebootURL)

	for try := 0; try < maxHttpRetry; try++ {
		rebootResponse, err := client.Get(rebootURL)
		if err != nil {
			if try < maxHttpRetry {
				continue
			}
			log.Println("    -> error:", err)
			os.Exit(1)
		}
		log.Println("  -> get: done")
		defer rebootResponse.Body.Close()

		log.Println("  -> read response:")
		rebootPage, err := ioutil.ReadAll(rebootResponse.Body)
		if err != nil {
			if try < maxHttpRetry {
				continue
			}
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
		break
	}

	log.Println("-> getDSToken: done: ", dsToken)
}

func reboot() {
	log.Println("-> reboot:")

	rebootActionURL := *modemURL + "/cgi-bin/reboot_action.cgi"
	rebootActionPayload := strings.NewReader(fmt.Sprintf(`waiting_action=1&DSToken=%s`, dsToken))

	log.Println("  -> post: waiting_action 1", rebootActionURL)
	response, err := client.Post(rebootActionURL, "application/x-www-form-urlencoded", rebootActionPayload)
	if err != nil {
		log.Println("    -> error:", err)
		os.Exit(1)
	}
	log.Println("  -> post: waiting_action 1 done")
	b, _ := ioutil.ReadAll(response.Body)
	log.Println(string(b))
	response.Body.Close()

	rebootActionPayload = strings.NewReader(fmt.Sprintf(`waiting_action=0&DSToken=%s`, dsToken))
	log.Println("  -> post: waiting_action 0", rebootActionURL)
	response, err = client.Post(rebootActionURL, "application/x-www-form-urlencoded", rebootActionPayload)
	if err != nil {
		if err, ok := err.(net.Error); !ok || !err.Timeout() {
			log.Println("    -> error:", err)
			os.Exit(1)
		}
	}

	log.Println("  -> post: waiting_action 0 done")
	if response != nil && response.Body != nil {
		b, _ := ioutil.ReadAll(response.Body)
		log.Println(string(b))
		response.Body.Close()
	}

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
	time.Sleep(5 * time.Second)
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
