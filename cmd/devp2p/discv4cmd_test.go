// Copyright 2019 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"ethereum/rpc-network/p2p/discover"
	"ethereum/rpc-network/p2p/enode"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"os"
	"testing"
	"time"
)

func init() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))
}

func TestCrawl(t *testing.T) {
	nodesFile := "11.json"
	var inputSet nodeSet

	var config discover.Config
	config.PrivateKey, _ = crypto.GenerateKey()

	dbpath := ""
	db, err := enode.OpenDB(dbpath)
	if err != nil {
		exit(err)
	}
	ln := enode.NewLocalNode(db, config.PrivateKey)

	socket := listen(ln, "")
	disc, err := discover.ListenV4(socket, ln, config)
	if err != nil {
		exit(err)
	}

	defer disc.Close()
	c := newCrawler(inputSet, disc, disc.RandomNodes())
	c.revalidateInterval = 10 * time.Minute
	output := c.run(time.Minute * 30)
	writeNodesJSON(nodesFile, output)
}
