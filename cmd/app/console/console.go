package console

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	asciiArt "github.com/common-nighthawk/go-figure"
	"gopkg.in/yaml.v3"
)

type question struct {
	prompt                 string
	validResponseRegex     string
	invalidResponseMessage string
	defaultResponse        string
}

var reader = bufio.NewReader(os.Stdin)
var yesNoRegexString = "^(y|Y|n|N)$"

var isYesRegex = regexp.MustCompile("y|Y")
var isNoRegex = regexp.MustCompile("n|N")
var yesNoInvalidResponse = "Please respond with y or n"

func RunWizard() {
	asciiArt.NewFigure("Tesla-GeoGDO Config Wizard", "", false).Print()

	config := map[string]interface{}{}
	response := promptUser(
		question{
			prompt:                 "\n\nWould you like to use the wizard to generate your config file? [Y|n]",
			validResponseRegex:     "^(y|Y|n|N)$",
			invalidResponseMessage: "Please respond with y or n",
			defaultResponse:        "y",
		},
	)
	if match, _ := regexp.MatchString("n|N|No|no|NO", response); match {
		return
	}

	config["global"] = runGlobalPrompts()
	config["garage_doors"] = runGarageDoorsPrompts()

	var b bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&b)
	yamlEncoder.SetIndent(2)
	yamlEncoder.Encode(config)
	yamlString := b.String()
	fmt.Print("\n\n####################### CONFIG FILE #######################")
	fmt.Print("\n\n" + yamlString + "\n\n")
	fmt.Print("##################### END CONFIG FILE #####################\n\n")

	asciiArt.NewFigure("Config Wizard Complete", "", false)

	fmt.Println("Congratulations on completing the config wizard. You can view your generated config above. You can choose to save this file automatically now, or you can copy and paste the contents above into your config file location.")
	response = promptUser(question{
		prompt:                 "Save file now? [Y|n]",
		validResponseRegex:     yesNoRegexString,
		invalidResponseMessage: yesNoInvalidResponse,
		defaultResponse:        "y",
	})
	if isNoRegex.MatchString(response) {
		return
	}
	filePath := promptUser(question{
		prompt:             "Where should the config file be saved? (remember, if running in a container, file path must be relative to container's file system) [/app/config/config.yml]",
		validResponseRegex: ".*",
		defaultResponse:    "/app/config/config.yml",
	})
	err := os.WriteFile(filePath, b.Bytes(), 0644)
	if err != nil {
		fmt.Printf("ERROR: Unable to write file to %s", filePath)
	}
}

func runGlobalPrompts() interface{} {
	asciiArt.NewFigure("Global Config", "", false).Print()

	response := promptUser(
		question{
			prompt:             "\nWhat is the DNS or IP of your tracker's MQTT broker (e.g. teslamate)? Note - if running in a container, you should not use localhost or 127.0.0.1",
			validResponseRegex: ".+",
		},
	)
	tracker_connection := map[string]interface{}{
		"host": response,
	}
	response = promptUser(
		question{
			prompt:                 "What is the port of your tracker's MQTT broker (e.g. teslamate)? [1883]",
			validResponseRegex:     "^\\d{1,5}$",
			invalidResponseMessage: "Please enter a number between 1 and 65534",
			defaultResponse:        "1883",
		},
	)
	port, _ := strconv.ParseFloat(response, 64)
	tracker_connection["port"] = port
	response = promptUser(
		question{
			prompt:             "Please enter the MQTT client ID to connect to the broker (can be left blank to auto generate at runtime): []",
			validResponseRegex: ".*",
		},
	)
	if response != "" {
		tracker_connection["client_id"] = response
	}
	response = promptUser(
		question{
			prompt:             "If your MQTT broker requires authentication, please enter the username: []",
			validResponseRegex: ".*",
		},
	)
	if response != "" {
		tracker_connection["user"] = response
	}
	response = promptUser(
		question{
			prompt:             "If your MQTT broker requires authentication, please enter the password (your typing will not be masked!); you can also enter this later: []",
			validResponseRegex: ".*",
		},
	)
	if response != "" {
		tracker_connection["pass"] = response
	}
	response = promptUser(
		question{
			prompt:                 "Does your broker require TLS? [y|N]",
			validResponseRegex:     "^(y|Y|n|N)$",
			invalidResponseMessage: "Please respond with y or n",
			defaultResponse:        "n",
		},
	)
	if match, _ := regexp.MatchString("y|Y", response); match {
		tracker_connection["use_tls"] = true
		response = promptUser(
			question{
				prompt:                 "Do you want to skip TLS verification (useful for self-signed certificates)? [y|N]",
				validResponseRegex:     "^(y|Y|n|N)$",
				invalidResponseMessage: "Please respond with y or n",
				defaultResponse:        "n",
			},
		)
		if match, _ := regexp.MatchString("y|Y", response); match {
			tracker_connection["skip_tls_verify"] = true
		}
	}

	tracker_mqtt_settings := map[string]interface{}{
		"connection": tracker_connection,
	}

	global_config := map[string]interface{}{
		"tracker_mqtt_settings": tracker_mqtt_settings,
	}

	response = promptUser(
		question{
			prompt:                 "Set the number (in minutes) that there should be a global cooldown for each door. This prevents any door from being operated for a set time after any previous operation. []",
			validResponseRegex:     "^(\\d+)?$",
			invalidResponseMessage: "Please enter a valid number (in minutes)",
		},
	)
	if len(response) > 0 {
		cooldown, _ := strconv.Atoi(response)
		global_config["cooldown"] = cooldown
	}
	return global_config
}

func runGarageDoorsPrompts() []interface{} {
	asciiArt.NewFigure("Garage Doors", "", false).Print()

	fmt.Print("\nWe will now configure one or more garage doors, which will include geofences, openers, and trackers\n\n")
	garage_doors := []interface{}{}
	re := regexp.MustCompile("n|N")

	for {
		garage_door := map[string]interface{}{}
		var response string
		response = promptUser(
			question{
				prompt:                 "What type of geofence would you like to configure for this garage door (more doors can be added later)?  [c|t|p]\nc: circular\nt: teslamate\np: polygon",
				validResponseRegex:     "^(c|t|p|C|T|P)$",
				invalidResponseMessage: "Please enter c (for circular), t (for teslamate), or p (for polygon)",
			},
		)
		switch response {
		case "c":
			garage_door["geofence"] = runCircularGeofencePrompts()
		case "t":
			garage_door["geofence"] = runTeslamateGeofencePrompts()
		case "p":
			garage_door["geofence"] = runPolygonGeofencePrompts()
		}

		response = promptUser(question{
			prompt:                 "\nWhat type of garage door opener would you like to configure for this garage door? [ha|hb|r|h|m]\nha: Home Assistant\nhb: Homebridge\nr: ratgdo (MQTT firmware only; for ESP Home, control via Home Assistant or Homebridge\nh: Generic HTTP\nm: Generic MQTT",
			validResponseRegex:     "^(ha|hb|r|h|m)$",
			invalidResponseMessage: "Please enter ha (for Home Assistant), hb (for Homebridge), r (for ratgdo), h (for generic HTTP), or m (for generic MQTT)",
		})

		switch response {
		case "ha":
			garage_door["opener"] = runHomeAssistantOpenerPrompts()
		case "hb":
			garage_door["opener"] = runHomebridgeOpenerPrompts()
		case "r":
			garage_door["opener"] = runRatgdoOpenerPrompts()
		case "h":
			garage_door["opener"] = runHttpOpenerPrompts()
		case "m":
			garage_door["opener"] = runMqttOpenerPrompts()
		}

		garage_door["trackers"] = runTrackerPrompts()

		garage_doors = append(garage_doors, garage_door)

		asciiArt.NewFigure("Garage Door Complete", "", false).Print()

		response = promptUser(question{
			prompt:                 "\nWould you like to configure another garage door? [y|n]",
			validResponseRegex:     "^(y|Y|n|N)$",
			invalidResponseMessage: "Please respond with y or n",
		})
		if re.MatchString(response) {
			break
		}
	}

	fmt.Println("Configuring garage doors is complete!")
	return garage_doors
}

func promptUser(q question) string {
	fmt.Println(q.prompt)
	response := readResponse()
	if len(response) == 0 && q.defaultResponse != "" {
		return q.defaultResponse
	}
	match, _ := regexp.MatchString(q.validResponseRegex, response)
	if !match {
		fmt.Println(q.invalidResponseMessage)
		return promptUser(q)
	}
	return response
}

func readResponse() string {
	text, _ := reader.ReadString('\n')
	text = strings.Replace(text, "\n", "", -1)
	text = strings.Replace(text, "\r", "", -1)
	return text
}
