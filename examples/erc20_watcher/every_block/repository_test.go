// Copyright 2018 Vulcanize
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package every_block_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vulcanize/vulcanizedb/examples/erc20_watcher/every_block"
	"github.com/vulcanize/vulcanizedb/examples/test_helpers"
	"github.com/vulcanize/vulcanizedb/pkg/core"
	"github.com/vulcanize/vulcanizedb/pkg/datastore/postgres"
	"github.com/vulcanize/vulcanizedb/pkg/datastore/postgres/repositories"
	"math/rand"
)

var db *postgres.DB

var _ = Describe("ERC20 Token Repository", func() {
	var blockId int64
	var blockNumber int64
	var repository every_block.TokenSupplyRepository
	var blockRepository repositories.BlockRepository

	BeforeEach(func() {
		db = test_helpers.CreateNewDatabase()
		repository = every_block.TokenSupplyRepository{DB: db}
		_, err := db.Query(`DELETE FROM token_supply`)
		Expect(err).NotTo(HaveOccurred())

		blockRepository = *repositories.NewBlockRepository(db)
		blockNumber = rand.Int63()
		blockId = createBlock(blockNumber, blockRepository)
	})

	Describe("Create", func() {
		It("creates a token supply record", func() {
			address := "abc"
			supply := supplyModel(blockNumber, address, "100")
			err := repository.Create(supply)
			Expect(err).NotTo(HaveOccurred())

			dbResult := TokenSupplyDBRow{}
			expectedTokenSupply := TokenSupplyDBRow{
				Supply:       int64(100),
				BlockID:      blockId,
				TokenAddress: address,
			}

			var count int
			err = repository.DB.QueryRowx(`SELECT count(*) FROM token_supply`).Scan(&count)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(1))

			err = repository.DB.QueryRowx(`SELECT * FROM token_supply`).StructScan(&dbResult)
			Expect(err).NotTo(HaveOccurred())
			Expect(dbResult.Supply).To(Equal(expectedTokenSupply.Supply))
			Expect(dbResult.BlockID).To(Equal(expectedTokenSupply.BlockID))
			Expect(dbResult.TokenAddress).To(Equal(expectedTokenSupply.TokenAddress))
		})

		It("returns an error if fetching the block's id from the database fails", func() {
			errorSupply := supplyModel(-1, "", "")
			err := repository.Create(errorSupply)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("sql"))
			Expect(err.Error()).To(ContainSubstring("block number -1"))
		})

		It("returns an error inserting the token_supply fails", func() {
			errorSupply := supplyModel(blockNumber, "", "")
			err := repository.Create(errorSupply)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("pq"))
			Expect(err.Error()).To(ContainSubstring("token_supply for block number"))
		})
	})

	Describe("MissingBlocks", func() {
		It("returns the block numbers for which an associated TokenSupply record hasn't been created", func() {
			createTokenSupplyFor(repository, blockNumber)
			newBlockNumber := blockNumber + 1
			createBlock(newBlockNumber, blockRepository)
			blocks, err := repository.MissingBlocks(blockNumber, newBlockNumber)
			anotherNewBlockNumber := newBlockNumber + 1
			createBlock(anotherNewBlockNumber, blockRepository)

			Expect(blocks).To(ConsistOf(newBlockNumber))
			Expect(err).NotTo(HaveOccurred())
		})

		It("does not return numbers that already have an associated TokenSupply record", func() {
			createTokenSupplyFor(repository, blockNumber)
			blocks, err := repository.MissingBlocks(blockNumber, blockNumber)

			Expect(blocks).To(BeEmpty())
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("deletes the token supply record when the associated block is deleted", func() {
		err := repository.Create(every_block.TokenSupply{BlockNumber: blockNumber, Value: "0"})
		Expect(err).NotTo(HaveOccurred())

		var count int
		err = repository.DB.QueryRowx(`SELECT count(*) FROM token_supply`).Scan(&count)
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(Equal(1))

		_, err = db.Query(`DELETE FROM blocks`)
		Expect(err).NotTo(HaveOccurred())

		err = repository.DB.QueryRowx(`SELECT count(*) FROM token_supply`).Scan(&count)
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(Equal(0))
	})
})

type TokenSupplyDBRow struct {
	ID           int64
	Supply       int64
	BlockID      int64  `db:"block_id"`
	TokenAddress string `db:"token_address"`
}

func supplyModel(blockNumber int64, tokenAddress string, supplyValue string) every_block.TokenSupply {
	return every_block.TokenSupply{
		Value:        supplyValue,
		TokenAddress: tokenAddress,
		BlockNumber:  int64(blockNumber),
	}
}

func createTokenSupplyFor(repository every_block.TokenSupplyRepository, blockNumber int64) {
	err := repository.Create(every_block.TokenSupply{BlockNumber: blockNumber, Value: "0"})
	Expect(err).NotTo(HaveOccurred())
}

func createBlock(blockNumber int64, repository repositories.BlockRepository) (blockId int64) {
	blockId, err := repository.CreateOrUpdateBlock(core.Block{Number: blockNumber})
	Expect(err).NotTo(HaveOccurred())

	return blockId
}
