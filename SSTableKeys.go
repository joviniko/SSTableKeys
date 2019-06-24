package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"sort"

	"github.com/golang/leveldb/table"
)

const majorVersionNumber = 2

func readIndexFile(filename string, protocolSetCount map[int]int, portSetCount map[int]int, ipv4SetCount map[string]int, ipv6SetCount map[string]int) error {

	fh, fhErr := os.Open(filename)

	if fhErr != nil {
		return fmt.Errorf("invalid file %q. %v", filename, fhErr)
	}
	ss := table.NewReader(fh, nil)
	if versions, err := ss.Get([]byte{0}, nil); err != nil {
		return fmt.Errorf("invalid index file %q missing versions record: %v", filename, err)
	} else if len(versions) != 8 {
		return fmt.Errorf("invalid index file %q invalid versions record: %v", filename, versions)
	} else if major := binary.BigEndian.Uint32(versions[:4]); major != majorVersionNumber {
		return fmt.Errorf("invalid index file %q: version mismatch, want %d got %d", filename, majorVersionNumber, major)
	}

	iter := ss.Find([]byte{}, nil)

	for iter.Next() {
		foundKey := iter.Key()
		ttype := foundKey[0]

		if ttype == 1 {
			proto := int(foundKey[1])
			protocolSetCount[proto] += len(iter.Value()) / 4
		} else if ttype == 2 {
			port := int(binary.BigEndian.Uint16([]byte{foundKey[1], foundKey[2]}))
			portSetCount[port] += len(iter.Value()) / 4
		} else if ttype == 4 {
			ipv4 := net.IP{foundKey[1],
				foundKey[2],
				foundKey[3],
				foundKey[4]}
			ipv4SetCount[ipv4.String()] += len(iter.Value()) / 4
		} else if ttype == 6 {
			ipv6 := net.IP{
				foundKey[1], foundKey[2], foundKey[3], foundKey[4],
				foundKey[5], foundKey[6], foundKey[7], foundKey[8],
				foundKey[9], foundKey[10], foundKey[11], foundKey[12],
				foundKey[13], foundKey[14], foundKey[15], foundKey[16],
			}
			ipv6SetCount[ipv6.String()] += len(iter.Value()) / 4
		}
	}
	iter.Close()
	fh.Close()

	return nil
}

func main() {
	if len(os.Args) != 2 && len(os.Args) != 4 {
		os.Exit(1)
	}

	protocolSetCount := make(map[int]int)
	portSetCount := make(map[int]int)
	ipv4SetCount := make(map[string]int)
	ipv6SetCount := make(map[string]int)

	folderPath := os.Args[1]
	startDate := ""
	endDate := ""

	inputTimestampArgs := regexp.MustCompile(`^\d{10}$`)

	if len(os.Args) == 4 {
		if !inputTimestampArgs.MatchString(os.Args[2]) || !inputTimestampArgs.MatchString(os.Args[3]) {
			log.Fatal("wrong timestamp input")
		}

		startDate = os.Args[2]
		endDate = os.Args[3]

		if endDate <= startDate {
			log.Fatal("start timestamp needs to be smaller than end timestamp")
		}
	}

	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		log.Fatal(err)
	}

	re := regexp.MustCompile(`^\d{16}$`)

	for _, file := range files {
		fileName := file.Name()
		if !re.MatchString(fileName) {
			continue
		}

		if startDate != "" && endDate != "" {
			fileNameShort := fileName[:10]
			if fileNameShort < startDate || endDate < fileNameShort {
				continue
			}
		}

		err := readIndexFile(fmt.Sprintf("%s/%s", folderPath, fileName), protocolSetCount, portSetCount, ipv4SetCount, ipv6SetCount)
		if err != nil {
			continue
		}
	}

	protocolsOut := ""
	if len(protocolSetCount) > 0 {
		sortedProtocols := make([]int, 0, len(protocolSetCount))
		for protocol := range protocolSetCount {
			sortedProtocols = append(sortedProtocols, protocol)
		}
		sort.Ints(sortedProtocols)

		for _, key := range sortedProtocols {
			protocolsOut += fmt.Sprintf(`"%d":%d,`, key, protocolSetCount[key])
		}
		protocolsOut = protocolsOut[:len(protocolsOut)-1]
	}

	portsOut := ""
	if len(portSetCount) > 0 {
		sortedPorts := make([]int, 0, len(portSetCount))
		for port := range portSetCount {
			sortedPorts = append(sortedPorts, port)
		}
		sort.Ints(sortedPorts)

		for _, key := range sortedPorts {
			portsOut += fmt.Sprintf(`"%d":%d,`, key, portSetCount[key])
		}
		portsOut = portsOut[:len(portsOut)-1]
	}

	ipv4Out := ""
	if len(ipv4SetCount) > 0 {
		sortedIPv4 := make([]net.IP, 0, len(ipv4SetCount))
		for ip := range ipv4SetCount {
			sortedIPv4 = append(sortedIPv4, net.ParseIP(ip))
		}
		sort.Slice(sortedIPv4, func(i, j int) bool {
			return bytes.Compare(sortedIPv4[i], sortedIPv4[j]) < 0
		})

		for _, key := range sortedIPv4 {
			ipString := key.String()
			ipv4Out += fmt.Sprintf(`"%s":%d,`, ipString, ipv4SetCount[ipString])
		}
		ipv4Out = ipv4Out[:len(ipv4Out)-1]
	}

	ipv6Out := ""
	if len(ipv6SetCount) > 0 {
		sortedIPv6 := make([]net.IP, 0, len(ipv6SetCount))
		for ip := range ipv6SetCount {
			sortedIPv6 = append(sortedIPv6, net.ParseIP(ip))
		}
		sort.Slice(sortedIPv6, func(i, j int) bool {
			return bytes.Compare(sortedIPv6[i], sortedIPv6[j]) < 0
		})

		for _, key := range sortedIPv6 {
			ipString := key.String()
			ipv6Out += fmt.Sprintf(`"%s":%d,`, ipString, ipv6SetCount[ipString])
		}
		ipv6Out = ipv6Out[:len(ipv6Out)-1]
	}

	out := fmt.Sprintf(`{"protocols":{%s},"ports":{%s},"ipv4":{%s},"ipv6":{%s}}`,
		protocolsOut,
		portsOut,
		ipv4Out,
		ipv6Out)

	fmt.Println(out)
}
