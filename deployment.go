package main

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"perun.network/channel-service/deployment"
	"perun.network/perun-ckb-backend/backend"
)

func GetSudt(config Config) (*deployment.SUDTInfo, error) {
	sudt := config.MigrationData.CellRecipes[3]
	if sudt.Name != "sudt" {
		return nil, fmt.Errorf("fourth cell recipe must be sudt, got %s", sudt.Name)
	}

	sudtScript := types.Script{
		CodeHash: types.HexToHash(sudt.DataHash),
		HashType: types.HashTypeData1,
		Args:     []byte{},
	}

	sudtCellDep := types.CellDep{
		OutPoint: &types.OutPoint{
			TxHash: types.HexToHash(sudt.TxHash),
			Index:  sudt.Index,
		},
		DepType: types.DepTypeCode,
	}

	return &deployment.SUDTInfo{
		Script:  &sudtScript,
		CellDep: &sudtCellDep,
	}, nil
}

func GetPubKey(key string) (secp256k1.PublicKey, error) {
	panic("Unimplemeted")
}

func GetDeployment(config Config, network types.Network) (backend.Deployment, deployment.SUDTInfo, error) {
	//read system scripts
	// read scripts for contract code
	pcts := config.MigrationData.CellRecipes[0]
	if pcts.Name != "pcts" {
		return backend.Deployment{}, deployment.SUDTInfo{}, fmt.Errorf("first cell recipe must be pcts, got %s", pcts.Name)
	}

	pcls := config.MigrationData.CellRecipes[1]
	if pcls.Name != "pcls" {
		return backend.Deployment{}, deployment.SUDTInfo{}, fmt.Errorf("second cell recipe must be pcls, got %s", pcls.Name)
	}

	pfls := config.MigrationData.CellRecipes[2]
	if pfls.Name != "pfls" {
		return backend.Deployment{}, deployment.SUDTInfo{}, fmt.Errorf("third cell recipe must be pfls, got %s", pfls.Name)
	}

	sudtInfo, err := GetSudt(config)
	if err != nil {
		return backend.Deployment{}, deployment.SUDTInfo{}, fmt.Errorf("getting sudt info: %w", err)
	}

	//NOTE: The SUDT lock-arg always contains a newline character at the end.
	hexString := strings.ReplaceAll(config.SUDTOwnerLockArg[2:], "\n", "")
	hexString = strings.ReplaceAll(hexString, "\r", "")
	hexString = strings.ReplaceAll(hexString, " ", "")
	sudtInfo.Script.Args, err = hex.DecodeString(hexString)

	if err != nil {
		return backend.Deployment{}, deployment.SUDTInfo{}, fmt.Errorf("decoding sudt owner lock arg: %w", err)
	}

	return backend.Deployment{
		Network: network,
		PCTSDep: types.CellDep{
			OutPoint: &types.OutPoint{
				TxHash: types.HexToHash(pcts.TxHash),
				Index:  config.MigrationData.CellRecipes[0].Index,
			},
			DepType: types.DepTypeCode,
		},
		PCLSDep: types.CellDep{
			OutPoint: &types.OutPoint{
				TxHash: types.HexToHash(pcls.TxHash),
				Index:  config.MigrationData.CellRecipes[1].Index,
			},
			DepType: types.DepTypeCode,
		},
		PFLSDep: types.CellDep{
			OutPoint: &types.OutPoint{
				TxHash: types.HexToHash(pfls.TxHash),
				Index:  config.MigrationData.CellRecipes[2].Index,
			},
			DepType: types.DepTypeCode,
		},
		PCTSCodeHash:    types.HexToHash(pcts.DataHash),
		PCTSHashType:    types.HashTypeData1,
		PCLSCodeHash:    types.HexToHash(pcls.DataHash),
		PCLSHashType:    types.HashTypeData1,
		PFLSCodeHash:    types.HexToHash(pfls.DataHash),
		PFLSHashType:    types.HashTypeData1,
		PFLSMinCapacity: deployment.PFLSMinCapacity,
		DefaultLockScript: types.Script{
			CodeHash: config.SystemScripts.Secp256k1Blake160SighashAll.ScriptID.CodeHash,
			HashType: config.SystemScripts.Secp256k1Blake160SighashAll.ScriptID.HashType,
			Args:     make([]byte, 32),
		},
		DefaultLockScriptDep: config.SystemScripts.Secp256k1Blake160SighashAll.CellDep,
		SUDTDeps: map[types.Hash]types.CellDep{
			sudtInfo.Script.Hash(): *sudtInfo.CellDep,
		},
		SUDTs: map[types.Hash]types.Script{
			sudtInfo.Script.Hash(): *sudtInfo.Script,
		},
	}, *sudtInfo, nil
}
