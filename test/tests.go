package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func CheckError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
	}
}

func main() {
	file, err := os.Open("test.log")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:5141")
	CheckError(err)

	LocalAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	CheckError(err)

	Conn, err := net.DialUDP("udp", LocalAddr, ServerAddr)
	CheckError(err)

	defer Conn.Close()

	for scanner.Scan() {
		buf := []byte(scanner.Text())
		//fmt.Printf("%s\n", buf)
		_, err := Conn.Write(buf)
		if err != nil {
			fmt.Println(err)
		}
	}

}
