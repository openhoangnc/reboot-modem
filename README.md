# reboot-modem
Reboot modem

```
Usage of reboot-modem
  -pwd string
        Password
  -url string
        URL (default "https://192.168.1.100:8080")
  -user string
        User (default "admin")
        
Example: ./reboot-modem -pwd ABC12345
```

This tool should work on H640W modem, not tested on other devices.

Q: Why this tool exists?

A: My ISP provided modem do not have schedule reboot function. I want this function.

Q: What next ?

A: Build this tool by Go, setup a crontab to execute it.
