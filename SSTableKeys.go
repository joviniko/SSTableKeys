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
	"strings"

	"github.com/golang/leveldb/table"
)

const majorVersionNumber = 2

type existingKeys struct {
	protocols []int
	ports     []int
	ipsv4     []net.IP
	ipsv6     []net.IP
}

func readIndexFile(filename string) (existingKeys, error) {
	returnVal := existingKeys{
		[]int{},
		[]int{},
		[]net.IP{},
		[]net.IP{},
	}

	fh, fhErr := os.Open(filename)

	if fhErr != nil {
		return returnVal, fmt.Errorf("invalid file %q. %v", filename, fhErr)
	}
	ss := table.NewReader(fh, nil)
	if versions, err := ss.Get([]byte{0}, nil); err != nil {
		return returnVal, fmt.Errorf("invalid index file %q missing versions record: %v", filename, err)
	} else if len(versions) != 8 {
		return returnVal, fmt.Errorf("invalid index file %q invalid versions record: %v", filename, versions)
	} else if major := binary.BigEndian.Uint32(versions[:4]); major != majorVersionNumber {
		return returnVal, fmt.Errorf("invalid index file %q: version mismatch, want %d got %d", filename, majorVersionNumber, major)
	}

	iter := ss.Find([]byte{}, nil)

	for iter.Next() {
		foundKey := iter.Key()
		ttype := foundKey[0]

		if ttype == 1 {
			proto := int(foundKey[1])
			returnVal.protocols = append(returnVal.protocols, proto)
		} else if ttype == 2 {
			port := []byte{foundKey[1], foundKey[2]}
			returnVal.ports = append(returnVal.ports, int(binary.BigEndian.Uint16(port)))
		} else if ttype == 4 {
			ipv4 := net.IP{foundKey[1],
				foundKey[2],
				foundKey[3],
				foundKey[4]}
			returnVal.ipsv4 = append(returnVal.ipsv4, ipv4)
		} else if ttype == 6 {
			ipv6 := net.IP{
				foundKey[1], foundKey[2], foundKey[3], foundKey[4],
				foundKey[5], foundKey[6], foundKey[7], foundKey[8],
				foundKey[9], foundKey[10], foundKey[11], foundKey[12],
				foundKey[13], foundKey[14], foundKey[15], foundKey[16],
			}

			returnVal.ipsv6 = append(returnVal.ipsv6, ipv6)
		}
	}
	iter.Close()
	fh.Close()

	return returnVal, nil
}

func main() {
	if len(os.Args) != 2 {
		os.Exit(1)
	}

	folderPath := os.Args[1]

	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		log.Fatal(err)
	}
	returnVal := existingKeys{
		[]int{},
		[]int{},
		[]net.IP{},
		[]net.IP{},
	}

	protocolSet := make(map[int]bool)
	portSet := make(map[int]bool)
	ipv4Set := make(map[string]bool)
	ipv6Set := make(map[string]bool)

	re := regexp.MustCompile(`\d{16}`)

	out := "test"
	for _, file := range files {
		fileName := file.Name()
		if !re.MatchString(fileName) {
			continue
		}

		returnValFile, err := readIndexFile(fmt.Sprintf("%s/%s", folderPath, fileName))
		if err != nil {
			continue
		}
		for _, item := range returnValFile.ipsv4 {
			ipv4Set[item.String()] = true
		}
		for _, item := range returnValFile.ipsv6 {
			ipv6Set[item.String()] = true
		}
		for _, item := range returnValFile.ports {
			portSet[item] = true
		}
		for _, item := range returnValFile.protocols {
			protocolSet[item] = true
		}
	}

	for k := range ipv4Set {
		returnVal.ipsv4 = append(returnVal.ipsv4, net.ParseIP(k))
	}
	for k := range ipv6Set {
		returnVal.ipsv6 = append(returnVal.ipsv6, net.ParseIP(k))
	}
	for k := range portSet {
		returnVal.ports = append(returnVal.ports, k)
	}
	for k := range protocolSet {
		returnVal.protocols = append(returnVal.protocols, k)
	}

	sortIPv4 := make([]net.IP, 0, len(returnVal.ipsv4))
	for _, ip := range returnVal.ipsv4 {
		sortIPv4 = append(sortIPv4, ip)
	}
	sort.Slice(sortIPv4, func(i, j int) bool {
		return bytes.Compare(sortIPv4[i], sortIPv4[j]) < 0
	})

	sortIPv6 := make([]net.IP, 0, len(returnVal.ipsv6))
	for _, ip := range returnVal.ipsv6 {
		sortIPv6 = append(sortIPv6, ip)
	}
	sort.Slice(sortIPv6, func(i, j int) bool {
		return bytes.Compare(sortIPv6[i], sortIPv6[j]) < 0
	})

	sort.Ints(returnVal.ports)
	sort.Ints(returnVal.protocols)
	ipv4Strings := []string{}
	for _, ipstr := range sortIPv4 {
		ipv4Strings = append(ipv4Strings, fmt.Sprintf(`"%s"`, ipstr))
	}
	ipv6Strings := []string{}
	for _, ipstr := range sortIPv6 {
		ipv6Strings = append(ipv6Strings, fmt.Sprintf(`"%s"`, ipstr))
	}

	out = fmt.Sprintf(`{"protocols":[%s],"ports":[%s],"ipv4":[%s],"ipv6":[%s]}`,
		strings.Trim(strings.Replace(fmt.Sprint(returnVal.protocols), " ", ",", -1), "[]"),
		strings.Trim(strings.Replace(fmt.Sprint(returnVal.ports), " ", ",", -1), "[]"),
		strings.Join(ipv4Strings, ","),
		strings.Join(ipv6Strings, ","))
	fmt.Println(out)
}
