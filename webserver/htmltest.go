package main
//package ads1x15_test

import (
	"fmt"
	"log"
	"math"
	//  "os/exec"
	"os"
	"time"
	"net/http"
    "html/template"
	"strconv"
    "gobot.io/x/gobot/platforms/raspi"
	"periph.io/x/periph/host"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/conn/physic"
    "periph.io/x/periph/experimental/devices/ads1x15"
	//"net"
)

type MyMessage struct {
    PageTitle string
    SprinklerMsg string
    MoistureMsg string
	Sprinkler string
	AutoOff string
}

var pi *raspi.Adaptor

var valueStr            string
var moistureLevel       float64
var channelNum ads1x15.Channel
var mL float64
var currentReading float64
var ReadingAtFullMoisture float64 = 1.20 // volts
var ReadingAtNoMoisture float64 = 3.00   // volts
var stopContinous  bool
var forcestop bool

func main() {

	pi = raspi.NewAdaptor()

	http.HandleFunc("/sprinkler/on", sprinklerOn)

	http.HandleFunc("/sprinkler/off", sprinklerOff)

	http.HandleFunc("/continousReading", continousReading)

 	http.HandleFunc("/stopContinousReading", stopContinousReading)
	
	 http.HandleFunc("/sprinkler/autooff", sprinklerAutoOff)

	//http.HandleFunc("/forceStoptheSprinkler", forceStoptheSprinkler)

	static := http.FileServer(http.Dir("../webcontent"))
	http.Handle("/", static)
	log.Printf("About to listen on 8081 to http://localhost:8081/")
	log.Fatal(http.ListenAndServe(":8081", nil))

}

func sprinklerAutoOff(w http.ResponseWriter, r *http.Request) {
	whichSprinkler := r.URL.Query()["which"]
	time.Sleep(5 * time.Second)
	switch whichSprinkler[0] {
		case "A":
			pi.DigitalWrite("31", 1)
		case "B":
			pi.DigitalWrite("35", 1)
		default:
			fmt.Println("Invalid Sprinkler")
	}
}

func getSensorReading(sprinklerNum string) float64 {

	// Make sure periph is initialized.
	if _, err := host.Init()
	err != nil {
		log.Fatal(err)
	}
	if sprinklerNum == "A" {
		channelNum = ads1x15.Channel0
	}else if sprinklerNum == "B" {
                channelNum = ads1x15.Channel1
	}else if sprinklerNum == "C" {
                channelNum = ads1x15.Channel2
	}else if sprinklerNum == "D" {
                channelNum = ads1x15.Channel3
	}
	// Open default I2C bus.
	bus, err := i2creg.Open("")
	if err != nil {
		log.Fatalf("failed to open I²C: %v", err)
	}
	defer bus.Close()

	// Create a new ADS1115 ADC.
	adc, err := ads1x15.NewADS1115(bus, &ads1x15.DefaultOpts)
	if err != nil {
		log.Fatalln(err)
	}

	// Obtain an analog pin from the ADC. PinForChannel(c Channel, maxVoltage physic.ElectricPotential, f physic.Frequency, q ConversionQuality)
        pin, err := adc.PinForChannel(channelNum, 5*physic.Volt, 1*physic.Hertz, ads1x15.SaveEnergy)
	fmt.Printf("value of  pin  :%v \n" , pin)
	if err != nil {
		log.Fatalln(err)
	}
	defer pin.Halt()


	// Single Reading
	reading, err := pin.Read()
	if err != nil {
		log.Fatalln(err)
 	}
	valueStr = reading.V.String()
	valueStr = valueStr[0:len(valueStr)-2]
	//moistureLevel,_ = strconv.ParseFloat(valueStr,2)
	currentReading,_ = strconv.ParseFloat(valueStr,2)
        fmt.Println(currentReading)
        fmt.Println(ReadingAtNoMoisture)
        fmt.Println(ReadingAtFullMoisture)
	mL = (currentReading - ReadingAtNoMoisture) / (ReadingAtFullMoisture - ReadingAtNoMoisture) * 100
	moistureLevel = math.Trunc(mL)
	return moistureLevel
}


func sprinklerOn(w http.ResponseWriter, r *http.Request) {
	var moistureLevel float64

	whichSprinkler := r.URL.Query()["which"]

	moistureLevel = getSensorReading(whichSprinkler[0])
	fmt.Println("Moisture Level :  ",moistureLevel, "%")
	if enoughMoisture(moistureLevel) == true {
		data := MyMessage {
                 PageTitle: "SPRINKLER ON",
                 SprinklerMsg: "Will attempt to turn on " + whichSprinkler[0],
	             MoistureMsg: "Enough Moisture. Sprinkler will not Turn On as percentage of moisture level is : " + fmt.Sprintf("%.2f", moistureLevel),
	    }
	    tmpl := template.Must(template.ParseFiles("myhtmlpage.html"))
	    tmpl.Execute(w, data)
	 return
	}

	switch whichSprinkler[0] {
		case "A":
			pi.DigitalWrite("31", 0)
		case "B":
			pi.DigitalWrite("35", 0)
		default:
			data := MyMessage {
				PageTitle: "SPRINKLER ERROR",
				SprinklerMsg: "",
				MoistureMsg: "Invalid name for sprinkler  : " + whichSprinkler[0],
			}
			tmpl := template.Must(template.ParseFiles("myhtmlpage.html"))
			tmpl.Execute(w, data)
			return
	}
	data := MyMessage {
		PageTitle: "SPRINKLER ON",
		SprinklerMsg: "Sprinkler " + whichSprinkler[0] + " is turned On, as there is no enough moisture level percentage of " + fmt.Sprintf("%.2f", moistureLevel),
		MoistureMsg: "" ,
		Sprinkler: whichSprinkler[0],
		AutoOff: "YES",	
	    }
	tmpl := template.Must(template.ParseFiles("myhtmlpage.html"))
	tmpl.Execute(w, data)
}       

func sprinklerOff(w http.ResponseWriter, r *http.Request) {
	var moistureLevel float64

	whichSprinkler := r.URL.Query()["which"]
	moistureLevel = getSensorReading(whichSprinkler[0])
	fmt.Println("Moisture Level : ",moistureLevel,"%")
/*	//if enoughMoisture(moistureLevel) == false {
		data := MyMessage {
			PageTitle: "SPRINKLER OFF",
			SprinklerMsg: "Will attempt to turn Offn " + whichSprinkler[0],
			MoistureMsg: "Not Enough Moisture. Sprinkler will not Turn Off as percentage of  moisture level is : " + fmt.Sprintf("%.2f", moistureLevel),
        }
        tmpl := template.Must(template.ParseFiles("myhtmlpage.html"))
         tmpl.Execute(w, data)
    // return
  */  

    switch whichSprinkler[0] {
		case "A":
			pi.DigitalWrite("31", 1)
		case "B":
			pi.DigitalWrite("35", 1)
		default:
			data := MyMessage {
			    	PageTitle: "SPRINKLER ERROR",
					SprinklerMsg: "",
					MoistureMsg: "Invalid name for sprinkler  : " + whichSprinkler[0],
			}
			tmpl := template.Must(template.ParseFiles("myhtmlpage.html"))
			tmpl.Execute(w, data)
			return
    }
	data := MyMessage {
		PageTitle: "SPRINKLER OFF",
		SprinklerMsg: "Sprinkler " + whichSprinkler[0] + " is turned Off, as there is enough moisture level percentage of " + fmt.Sprintf("%.2f", moistureLevel),
		MoistureMsg: "" ,
    }
	tmpl := template.Must(template.ParseFiles("myhtmlpage.html"))
        tmpl.Execute(w, data)  
}

func enoughMoisture(moistureLevel float64) bool {
	var rValue bool
	if moistureLevel < 70 {
		rValue = false
	}
	if moistureLevel > 40 {
		rValue = true
	}
	return rValue
}

func continousReading(w http.ResponseWriter, r *http.Request){

	go startContinousReading()
	
	time.Sleep(3 * time.Second)
	data := MyMessage {
		PageTitle: "CONTINOUS READING STARTED",
		SprinklerMsg: "Continous Reading Started",
		MoistureMsg: "" ,
    }
	tmpl := template.Must(template.ParseFiles("myhtmlpage2.html"))
    tmpl.Execute(w, data) 
	
}

func stopContinousReading(w http.ResponseWriter, r *http.Request){
	stopContinous = true
	data := MyMessage {
		PageTitle: " CONTINOUS READING STOPPED",
		SprinklerMsg: "Continous Reading for Sprinkler 'A' is stopped",
		MoistureMsg: "" ,
    }
	tmpl := template.Must(template.ParseFiles("myhtmlpage.html"))
   	tmpl.Execute(w, data) 
	
	err := os.Remove("../webcontent/contreadingdata.html")
	if err != nil {
        log.Fatalln(err)
    }
	
}

func startContinousReading(){

	stopContinous = false
        // Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Open default I²C bus.
	bus, err := i2creg.Open("")
	if err != nil {
		log.Fatalf("failed to open I²C: %v", err)
	}
	defer bus.Close()

	// Create a new ADS1115 ADC.
	adc, err := ads1x15.NewADS1115(bus, &ads1x15.DefaultOpts)
	if err != nil {
		log.Fatalln(err)
	}

	
	f, err := os.Create("../webcontent/contreadingdata.html")
	if err != nil {
        log.Fatalln(err)
    }
	
	/*
    defer f.Close()
	*/

    // Obtain an analog pin from the ADC.
	pin, err := adc.PinForChannel(ads1x15.Channel1, 5*physic.Volt, 1*physic.Hertz, ads1x15.SaveEnergy)
	if err != nil {
		log.Fatalln(err)
	}
    
	// Read values continuously from ADC.
	for {
		       
		c := pin.ReadContinuous()
		
		for reading := range c {
			valueStr = reading.V.String()
			valueStr = valueStr[0:len(valueStr)-2]
			currentReading,_ = strconv.ParseFloat(valueStr,2)
			mL = (currentReading - ReadingAtNoMoisture) / (ReadingAtFullMoisture - ReadingAtNoMoisture) * 100
			moistureLevel = math.Trunc(mL)
			dt := time.Now()
			// time.Sleep(5 * time.Second)
			fmt.Println(dt.Format("01-02-2006 15:04:05")," - Moisture Level : ",moistureLevel,"%")
			
			_, err2 := f.WriteString(dt.Format("01-02-2006 15:04:05") + " - Moisture Level : " + fmt.Sprintf("%.2f", moistureLevel) + "<BR>")
			if err2 != nil {
				log.Fatalln(err2)
			}
			
			if enoughMoisture(moistureLevel) == false {
				pi.DigitalWrite("35", 0)
				if stopContinous == true {
					fmt.Println("Inside Stopping")
					pi.DigitalWrite("35", 1)
					return
				}
			}
			if enoughMoisture(moistureLevel) == true {
				pi.DigitalWrite("35", 1)
			}
            if stopContinous == true {
                return
            }
		}
	}
}



func sleepforfewsecA() {
	
	sleepforsec:=true

	if sleepforsec == true {
		pi.DigitalWrite("31" , 1)

		}

}

func sleepforfewsecB() {

    sleepforsec:=true
	time.Sleep(5 * time.Second)

    if sleepforsec == true {
        pi.DigitalWrite("35" , 1)
    }

}

func turnOnMessage(w http.ResponseWriter) {

	fmt.Println("turnOnMessage ...")

    data := MyMessage {
		PageTitle: "SPRINKLER ON",
		SprinklerMsg: "My Message is Coming for On",
		MoistureMsg: "" ,

	    }
	fmt.Println("turnOnMessage 1...")
	tmpl := template.Must(template.ParseFiles("myhtmlpage.html"))
	fmt.Println("turnOnMessage 2...")
	tmpl.Execute(w, data)
	
}
