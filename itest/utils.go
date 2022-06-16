package itest

/*
 * Dual-licensed under Apache-2.0 and MIT.
 *
 * You can get a copy of the Apache License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * You can also get a copy of the MIT License at
 *
 * http://opensource.org/licenses/MIT
 *
 * @wcgcyx - https://github.com/wcgcyx
 */

import (
	"context"
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
	"time"
)

// initKey init and generate a key
func initKey(nodes []string) error {
	for _, node := range nodes {
		out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v fcr wallet generate 1", node)).Output()
		if err != nil {
			return err
		}
		key := string(out)
		out, err = exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v fcr wallet set 1 1 %v", node, key[:len(key)-1])).Output()
		if err != nil {
			return err
		}
		if string(out) != "Succeed\n" {
			return fmt.Errorf("failed with error %v", string(out))
		}
	}
	return nil
}

// setPolicy sets the default settlement policy of given nodes to be 24 hours.
//
// @input - nodes.
//
// @output - error.
func setPolicy(nodes []string) error {
	for _, node := range nodes {
		out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v fcr paynet policy settle set 1 default 24h", node)).Output()
		if err != nil {
			return err
		}
		if string(out) != "Succeed\n" {
			return fmt.Errorf("failed with error %v", string(out))
		}
		out, err = exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v fcr paynet policy reserve set 1 default -1", node)).Output()
		if err != nil {
			return err
		}
		if string(out) != "Succeed\n" {
			return fmt.Errorf("failed with error %v", string(out))
		}
	}
	return nil
}

// connect establishes p2p connection for given nodes.
//
// @input - from node, map of to nodes to number of channels, whether to serve the channel.
//
// @output - error.
func connect(from string, tos map[string]int, served bool) error {
	for to, num := range tos {
		out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v fcr system addr", to)).Output()
		if err != nil {
			return err
		}
		addr := strings.Split(string(out), " ")[0][1:]
		out, err = exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v fcr system connect %v", from, addr)).Output()
		if err != nil {
			return err
		}
		out, err = exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v fcr system publish", to)).Output()
		if err != nil {
			return err
		}
		if string(out) != "Succeed\n" {
			return fmt.Errorf("failed with error %v", string(out))
		}
		out, err = exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v fcr wallet get 1", to)).Output()
		if err != nil {
			return err
		}
		toAddr := strings.Split(string(out), " ")[0]
		// Create channel now.
		// Get paych offer
		out, err = exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v fcr paynet paych query 1 %v", from, toAddr)).Output()
		if err != nil {
			return err
		}
		offer := strings.Split(string(out), "\n")[5]
		offer = strings.Split(offer, ": ")[1]
		for i := 0; i < num; i++ {
			// Create paych
			out, err = exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v fcr paynet paych create %v 1000000000000000000", from, offer)).Output()
			if err != nil {
				return err
			}
			chAddr := strings.Split(string(out), "\n")[1]
			if served {
				// Serve payment channel
				out, err = exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v fcr paynet serving serve 1 %v %v 1 100", from, toAddr, chAddr)).Output()
				if err != nil {
					return err
				}
				if string(out) != "Succeed\n" {
					return fmt.Errorf("failed with error %v", string(out))
				}
			}
		}
	}
	return nil
}

// generate generates a piece, import and serve.
//
// @input - from node, number of files to generate.
//
// @output - error.
func generate(from string, num int) error {
	for i := 0; i < num; i++ {
		// Generate
		_, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v sh -c \"tr -dc A-Za-z0-9 </dev/urandom | head -c 10747904 > /app/testfile\"", from)).Output()
		if err != nil {
			return err
		}
		// Import
		out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v fcr cidnet piece import /app/testfile", from)).Output()
		if err != nil {
			return err
		}
		id := strings.Split(string(out), ": ")[1]
		id = strings.Split(id, "\n")[0]
		// Serve the file
		out, err = exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v fcr cidnet serving serve %v 1 1", from, id)).Output()
		if err != nil {
			return err
		}
		if string(out) != "Succeed\n" {
			return fmt.Errorf("failed with error %v", string(out))
		}
		// Remove copy
		_, err = exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v rm /app/testfile", from)).Output()
		if err != nil {
			return err
		}
	}
	return nil
}

// load will load imported pieces for given node.
//
// @input - from node.
//
// @output - imported pieces, error.
func load(from string) ([]string, error) {
	out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v fcr cidnet piece list", from)).Output()
	if err != nil {
		fmt.Println(string(out))
		return nil, err
	}
	ids := strings.Split(string(out), "\n")
	return ids[:len(ids)-2], nil
}

// retrieve will retrieve a number of files using fast-retrieve.
//
// @input - from address, total imported map, region 1 retrieval number, region 2 retrieval number, region 3 retrieval number.
//
// @output - error.
func retrieve(from string, totalImported map[int][]string, reg1 int, reg2 int, reg3 int) error {
	// Make a copy of the original imported.
	imported := make(map[int][]string)
	imported[0] = make([]string, 0)
	imported[1] = make([]string, 0)
	imported[2] = make([]string, 0)
	for _, val := range totalImported[0] {
		imported[0] = append(imported[0], val)
	}
	for _, val := range totalImported[1] {
		imported[1] = append(imported[1], val)
	}
	for _, val := range totalImported[2] {
		imported[2] = append(imported[2], val)
	}
	toRetrieve := make([]string, 0)
	// Shuffle imported.
	for i := 0; i < 3; i++ {
		for j := range imported[i] {
			k := rand.Intn(j + 1)
			imported[i][j], imported[i][k] = imported[i][k], imported[i][j]
		}
	}
	if reg1 > 0 && reg1 <= len(imported[0]) {
		toRetrieve = append(toRetrieve, imported[0][:reg1]...)
	}
	if reg2 > 0 && reg2 <= len(imported[1]) {
		toRetrieve = append(toRetrieve, imported[1][:reg2]...)
	}
	if reg3 > 0 && reg3 <= len(imported[2]) {
		toRetrieve = append(toRetrieve, imported[2][:reg3]...)
	}
	// Shuttle to retrieve
	indexes := make([]int, len(toRetrieve))
	for i := 0; i < len(toRetrieve); i++ {
		indexes[i] = i
	}
	for i := range indexes {
		j := rand.Intn(i + 1)
		indexes[i], indexes[j] = indexes[j], indexes[i]
	}
	failed := 0
	for _, index := range indexes {
		piece := toRetrieve[index]
		func(piece string, index int) {
			region := "unknown"
			if index < reg1 {
				region = "region 1"
			} else if index < reg1+reg2 {
				region = "region 2"
			} else if index < reg1+reg2+reg3 {
				region = "region 3"
			}
			out, _ := exec.CommandContext(context.Background(), "/bin/sh", "-c", fmt.Sprintf("docker exec %v fcr fast-retrieve %v /app/ 1 10000000000", from, piece)).Output()
			time.Sleep(1 * time.Second)
			_, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v ls -l /app/%v", from, piece)).Output()
			if err != nil {
				fmt.Printf("Retrieval of %v in %v from %v failed: cannot find piece locally %v\n", piece, region, from, string(out))
				failed += 1
				return
			} else {
				exec.Command("/bin/sh", "-c", fmt.Sprintf("docker exec %v rm /app/%v", from, piece)).Output()
			}
			fmt.Printf("Retrieval of %v in %v from %v succeed\n", piece, region, from)
		}(piece, index)
	}
	fmt.Printf("Retreival %v succeed\n", reg1+reg2+reg3-failed)
	if failed != 0 {
		return fmt.Errorf("Has failed retrieval")
	}
	return nil
}
