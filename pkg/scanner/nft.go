package scanner

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	// TODO: Replace with github.com/ethereum/go-ethereum when available
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func (s *Scanner) scanNFTHolders(contractAddr common.Address, currentBlock uint64) (map[string]*AssetHolder, error) {
	holders := make(map[string]*AssetHolder)

	// Load ABI
	nftABI, err := abi.JSON(strings.NewReader(erc721ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse NFT ABI: %w", err)
	}

	// Try to get total supply first
	totalSupply, err := s.getNFTTotalSupply(contractAddr, nftABI)
	if err == nil && totalSupply.Cmp(big.NewInt(0)) > 0 {
		log.Printf("Total supply: %s", totalSupply.String())

		// Scan by token ID
		for i := big.NewInt(0); i.Cmp(totalSupply) < 0; i.Add(i, big.NewInt(1)) {
			tokenID := new(big.Int).Set(i)

			owner, err := s.getNFTOwner(contractAddr, nftABI, tokenID)
			if err != nil || owner == (common.Address{}) {
				continue
			}

			// Get token URI for type detection
			tokenURI, _ := s.getNFTTokenURI(contractAddr, nftABI, tokenID)
			collectionType := s.determineNFTType(tokenID, tokenURI)

			if _, exists := holders[owner.Hex()]; !exists {
				holders[owner.Hex()] = &AssetHolder{
					Address:         owner,
					TokenIDs:        []*big.Int{},
					AssetType:       "NFT",
					CollectionType:  collectionType,
					StakingPower:    s.project.StakingPowers[collectionType],
					ChainName:       s.config.Chain,
					ContractAddress: contractAddr.Hex(),
					ProjectName:     s.config.ProjectName,
				}
			}

			holders[owner.Hex()].TokenIDs = append(holders[owner.Hex()].TokenIDs, tokenID)

			if len(holders)%100 == 0 {
				log.Printf("Scanned %d NFT holders...", len(holders))
			}
		}
	} else {
		// Fall back to event scanning
		log.Printf("Falling back to event scanning...")
		return s.scanNFTHoldersByEvents(contractAddr, currentBlock)
	}

	return holders, nil
}

func (s *Scanner) scanNFTHoldersByEvents(contractAddr common.Address, currentBlock uint64) (map[string]*AssetHolder, error) {
	holders := make(map[string]*AssetHolder)
	nftOwnership := make(map[string]common.Address) // tokenID -> current owner

	// Load ABI
	nftABI, err := abi.JSON(strings.NewReader(erc721ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse NFT ABI: %w", err)
	}

	// Calculate block range
	fromBlock := currentBlock - uint64(s.config.BlockRange)
	if fromBlock < 0 {
		fromBlock = 0
	}

	// Scan in chunks
	chunkSize := uint64(10000)

	for start := fromBlock; start < currentBlock; start += chunkSize {
		end := start + chunkSize - 1
		if end > currentBlock {
			end = currentBlock
		}

		log.Printf("Scanning blocks %d to %d...", start, end)

		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(start)),
			ToBlock:   big.NewInt(int64(end)),
			Addresses: []common.Address{contractAddr},
			Topics:    [][]common.Hash{{nftABI.Events["Transfer"].ID}},
		}

		logs, err := s.client.FilterLogs(context.Background(), query)
		if err != nil {
			log.Printf("Warning: Failed to get logs for blocks %d-%d: %v", start, end, err)
			continue
		}

		for _, vLog := range logs {
			if len(vLog.Topics) >= 4 {
				// from := common.HexToAddress(vLog.Topics[1].Hex()) // Not used yet
				to := common.HexToAddress(vLog.Topics[2].Hex())
				tokenID := new(big.Int).SetBytes(vLog.Topics[3].Bytes())

				// Update ownership
				if to == (common.Address{}) {
					// Token burned
					delete(nftOwnership, tokenID.String())
				} else {
					nftOwnership[tokenID.String()] = to
				}
			}
		}

		time.Sleep(100 * time.Millisecond) // Rate limiting
	}

	// Build holders map from ownership
	for tokenIDStr, owner := range nftOwnership {
		tokenID := new(big.Int)
		tokenID.SetString(tokenIDStr, 10)

		// Get token URI for type detection
		tokenURI, _ := s.getNFTTokenURI(contractAddr, nftABI, tokenID)
		collectionType := s.determineNFTType(tokenID, tokenURI)

		if _, exists := holders[owner.Hex()]; !exists {
			holders[owner.Hex()] = &AssetHolder{
				Address:         owner,
				TokenIDs:        []*big.Int{},
				AssetType:       "NFT",
				CollectionType:  collectionType,
				StakingPower:    s.project.StakingPowers[collectionType],
				ChainName:       s.config.Chain,
				ContractAddress: contractAddr.Hex(),
				ProjectName:     s.config.ProjectName,
			}
		}

		holders[owner.Hex()].TokenIDs = append(holders[owner.Hex()].TokenIDs, tokenID)
	}

	return holders, nil
}

func (s *Scanner) getNFTTotalSupply(contractAddr common.Address, abi abi.ABI) (*big.Int, error) {
	data, err := abi.Pack("totalSupply")
	if err != nil {
		return nil, err
	}

	msg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: data,
	}

	result, err := s.client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, err
	}

	var totalSupply *big.Int
	err = abi.UnpackIntoInterface(&totalSupply, "totalSupply", result)
	if err != nil {
		return nil, err
	}

	return totalSupply, nil
}

func (s *Scanner) getNFTOwner(contractAddr common.Address, abi abi.ABI, tokenID *big.Int) (common.Address, error) {
	data, err := abi.Pack("ownerOf", tokenID)
	if err != nil {
		return common.Address{}, err
	}

	msg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: data,
	}

	result, err := s.client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return common.Address{}, err
	}

	var owner common.Address
	err = abi.UnpackIntoInterface(&owner, "ownerOf", result)
	if err != nil {
		return common.Address{}, err
	}

	return owner, nil
}

func (s *Scanner) getNFTTokenURI(contractAddr common.Address, abi abi.ABI, tokenID *big.Int) (string, error) {
	data, err := abi.Pack("tokenURI", tokenID)
	if err != nil {
		return "", err
	}

	msg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: data,
	}

	result, err := s.client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return "", err
	}

	var tokenURI string
	err = abi.UnpackIntoInterface(&tokenURI, "tokenURI", result)
	if err != nil {
		return "", err
	}

	return tokenURI, nil
}

func (s *Scanner) determineNFTType(tokenID *big.Int, tokenURI string) string {
	uriLower := strings.ToLower(tokenURI)

	// Check type identifiers
	for collectionType, keywords := range s.project.TypeIdentifiers {
		for _, keyword := range keywords {
			if strings.Contains(uriLower, keyword) {
				return collectionType
			}
		}
	}

	// Default fallback based on project
	switch s.config.ProjectName {
	case "lux":
		if tokenID.Cmp(big.NewInt(1000)) < 0 {
			return "Validator"
		} else if tokenID.Cmp(big.NewInt(5000)) < 0 {
			return "Card"
		}
		return "Coin"
	case "zoo":
		return "Animal"
	case "spc":
		return "Pony"
	case "hanzo":
		return "AI"
	default:
		return "Unknown"
	}
}
