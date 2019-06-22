package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/golang/leveldb/table"
	"github.com/google/stenographer/filecache"
)

const majorVersionNumber = 2

func NewIndexFile(filename string, fc *filecache.Cache) (string, error) {
	ss := table.NewReader(fc.Open(filename), nil)
	if versions, err := ss.Get([]byte{0}, nil); err != nil {
		return "", fmt.Errorf("invalid index file %q missing versions record: %v", filename, err)
	} else if len(versions) != 8 {
		return "", fmt.Errorf("invalid index file %q invalid versions record: %v", filename, versions)
	} else if major := binary.BigEndian.Uint32(versions[:4]); major != majorVersionNumber {
		return "", fmt.Errorf("invalid index file %q: version mismatch, want %d got %d", filename, majorVersionNumber, major)
	}

	iter := ss.Find([]byte{}, nil)
	protocols := []string{}
	ports := []string{}
	ips_v4 := []string{}
	ips_v6 := []string{}
	for iter.Next() {
		found_key := iter.Key()
		ttype := found_key[0]

		if ttype == 1 {
			proto := int(found_key[1])
			protocols = append(protocols, fmt.Sprintf("%d", proto))
		} else if ttype == 2 {
			port := (int(found_key[1]) * 1000) + int(found_key[2])
			ports = append(ports, fmt.Sprintf("%d", port))
		} else if ttype == 4 {
			ipv4 := net.IP{found_key[1],
				found_key[2],
				found_key[3],
				found_key[4]}

			ips_v4 = append(ips_v4, ipv4.String())
		} else if ttype == 6 {
			ipv6 := net.IP{found_key[1],
				found_key[2],
				found_key[3],
				found_key[4],
				found_key[5],
				found_key[6],
				found_key[7],
				found_key[8],
				found_key[9],
				found_key[10],
				found_key[11],
				found_key[12],
				found_key[13],
				found_key[14],
				found_key[15],
				found_key[16]}

			ips_v6 = append(ips_v6, ipv6.String())
		}
	}
	iter.Close()
	out := fmt.Sprintf(`{"protocols":[%s],"ports":[%s],"ipv4":[%s],"ipv6":[%s]}`,
		strings.Join(protocols, ","), strings.Join(ports, ","), strings.Join(ips_v4, ","), strings.Join(ips_v6, ","))
	return out, nil
}

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}
	filename := os.Args[1]
	output, err := NewIndexFile(filename, filecache.NewCache(1))
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(output)
}
