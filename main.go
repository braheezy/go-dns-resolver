package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"strings"
)

// https://implement-dns.wizardzines.com/book/intro.html

type DNSHeader struct {
	id          uint16
	flags       uint16
	questions   uint16
	answers     uint16
	authorities uint16
	additional  uint16
}

type DNSQuestion struct {
	name  []byte
	type_ uint16
	class uint16
}

// https://datatracker.ietf.org/doc/html/rfc1035#section-4.1.3
type DNSRecord struct {
	name    []byte
	type_   uint16
	class   uint16
	ttl     uint32
	dataLen uint16
	data    []byte
}

type DNSPacket struct {
	header      DNSHeader
	questions   []DNSQuestion
	answers     []DNSRecord
	authorities []DNSRecord
	additional  []DNSRecord
}

var TYPE_A uint16 = 1
var CLASS_IN uint16 = 1

// Convert the struct into a byte string
func headerToBytes(h *DNSHeader) []byte {
	byteData := make([]byte, 12)

	binary.BigEndian.PutUint16(byteData[0:2], h.id)
	binary.BigEndian.PutUint16(byteData[2:4], h.flags)
	binary.BigEndian.PutUint16(byteData[4:6], h.questions)
	binary.BigEndian.PutUint16(byteData[6:8], h.answers)
	binary.BigEndian.PutUint16(byteData[8:10], h.authorities)
	binary.BigEndian.PutUint16(byteData[10:12], h.additional)

	return byteData
}

func questionToBytes(q *DNSQuestion) []byte {
	byteData := make([]byte, len(q.name)+6)
	copy(byteData[0:], q.name)
	binary.BigEndian.PutUint16(byteData[len(q.name):], q.type_)
	binary.BigEndian.PutUint16(byteData[len(q.name)+2:], q.class)

	return byteData
}

// Encode the domain name
func encodeDNSName(domain_name string) []byte {
	// To obtain the encoding, split the domain name into parts then prepend each part with its length
	var result []byte

	for _, part := range strings.Split(domain_name, ".") {
		result = append(result, byte(len(part)))
		result = append(result, part...)
	}

	result = append(result, 0)

	return result
}

func buildQuery(domainName string, recordType uint16) []byte {
	encodedName := encodeDNSName(domainName)
	id := rand.Intn(65535)
	RECURSION_DESIRED := 1 << 8
	header := DNSHeader{
		id:        uint16(id),
		questions: 1,
		flags:     uint16(RECURSION_DESIRED),
	}
	question := DNSQuestion{
		name:  encodedName,
		type_: recordType,
		class: uint16(CLASS_IN),
	}

	headerBytes := headerToBytes(&header)
	questionBytes := questionToBytes(&question)

	query := make([]byte, len(headerBytes)+len(questionBytes))
	copy(query, headerBytes)
	copy(query[len(headerBytes):], questionBytes)

	return query
}

func parseHeader(buffer *bytes.Buffer) DNSHeader {
	var header DNSHeader

	binary.Read(buffer, binary.BigEndian, &header.id)
	binary.Read(buffer, binary.BigEndian, &header.flags)
	binary.Read(buffer, binary.BigEndian, &header.questions)
	binary.Read(buffer, binary.BigEndian, &header.answers)
	binary.Read(buffer, binary.BigEndian, &header.authorities)
	binary.Read(buffer, binary.BigEndian, &header.additional)

	return header
}

func parseQuestion(buffer *bytes.Buffer) DNSQuestion {
	var question DNSQuestion

	name, _ := parseName(buffer)
	question.name = []byte(name)

	binary.Read(buffer, binary.BigEndian, &question.type_)
	binary.Read(buffer, binary.BigEndian, &question.class)
	return question
}

// Parse the "question" in the response to find the domain name.
// This is tricky because of DNS compression
// https://datatracker.ietf.org/doc/html/rfc1035#section-4.1.4
func parseName(buffer *bytes.Buffer) (string, error) {
	var name bytes.Buffer
	for {
		lengthByte, err := buffer.ReadByte()
		if err != nil {
			return "", err
		}

		// Check if the length byte is a pointer
		if lengthByte&0xC0 == 0xC0 {
			pointerByte, err := buffer.ReadByte()
			if err != nil {
				return "", err
			}

			// Calculate the offset from the pointer
			offset := int(((lengthByte & 0x3F) << 8) | pointerByte)

			// Read the name from the offset position
			originalPos := buffer.Len()
			buffer.Truncate(offset)
			namePart, err := parseName(buffer)
			if err != nil {
				return "", err
			}
			buffer.Truncate(originalPos)

			// Append the parsed name part to the complete name
			name.WriteString(namePart)
			name.WriteString(".")
			break
		}

		// Read the label
		label := make([]byte, lengthByte)
		_, err = buffer.Read(label)
		if err != nil {
			return "", err
		}

		// Append the label to the complete name
		name.Write(label)
		name.WriteString(".")

		// Exit the loop if the length byte is zero
		if lengthByte == 0 {
			break
		}
	}

	return strings.TrimSuffix(name.String(), "."), nil
}

func parseDNSRecord(buffer *bytes.Buffer) (*DNSRecord, error) {
	name, err := parseName(buffer)
	if err != nil {
		return nil, err
	}

	var record DNSRecord
	err = binary.Read(buffer, binary.BigEndian, &record.type_)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buffer, binary.BigEndian, &record.class)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buffer, binary.BigEndian, &record.ttl)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buffer, binary.BigEndian, &record.dataLen)
	if err != nil {
		return nil, err
	}

	record.data = make([]byte, record.dataLen)
	_, err = buffer.Read(record.data)
	if err != nil {
		return nil, err
	}

	record.name = []byte(name)

	return &record, nil
}

func main() {
	serverAddr, err := net.ResolveUDPAddr("udp", "8.8.8.8:53")
	if err != nil {
		panic(err)
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		panic(err)
	}

	query := buildQuery("www.example.com", TYPE_A)
	_, err = conn.Write(query)
	if err != nil {
		panic(err)
	}

	// Read back the UDP response
	response := make([]byte, 512)
	_, err = conn.Read(response)
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully sent query")

	// Create a buffer from the response byte slice
	buffer := bytes.NewBuffer(response)

	// Parse the DNS header
	header := parseHeader(buffer)
	fmt.Println("DNS Header:")
	fmt.Printf("ID: %d\n", header.id)
	fmt.Printf("Flags: %d\n", header.flags)
	fmt.Printf("Questions: %d\n", header.questions)
	fmt.Printf("Answers: %d\n", header.answers)
	fmt.Printf("Authorities: %d\n", header.authorities)
	fmt.Printf("Additional: %d\n", header.additional)
	fmt.Println()

	// Parse the DNS question
	question := parseQuestion(buffer)
	fmt.Println("DNS Question:")
	fmt.Printf("Name: %s\n", question.name)
	fmt.Printf("Type: %d\n", question.type_)
	fmt.Printf("Class: %d\n", question.class)
	fmt.Println()

	buffer.Next(2)

	// Parse the DNS record(s)
	for i := 0; i < int(header.answers); i++ {
		record, _ := parseDNSRecord(buffer)
		fmt.Println("DNS Record:")
		fmt.Printf("Name: %s\n", record.name)
		fmt.Printf("Type: %d\n", record.type_)
		fmt.Printf("Class: %d\n", record.class)
		fmt.Printf("TTL: %d\n", record.ttl)
		fmt.Printf("Data Length: %d\n", record.dataLen)
		fmt.Printf("Data: %v\n", record.data)
		fmt.Println()
	}
}
