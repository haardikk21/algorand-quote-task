package main

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/algorand/go-algorand-sdk/transaction"

	"github.com/algorand/go-algorand-sdk/client/algod"
	"github.com/algorand/go-algorand-sdk/client/algod/models"
	"github.com/algorand/go-algorand-sdk/client/kmd"
	"github.com/algorand/go-algorand-sdk/types"
)

var (
	algodAddress          = os.Getenv("ALGOD_ADDRESS")
	algodToken            = os.Getenv("ALGOD_TOKEN")
	kmdAddress            = os.Getenv("KMD_ADDRESS")
	kmdToken              = os.Getenv("KMD_TOKEN")
	note                  = []byte("Easy choices, hard life. Hard choices, easy life.") // Note to attach to transaction
	amount         uint64 = 1000                                                        // amount to send
	walletName            = "testwallet2"
	walletPassword        = "testpassword"
)

func main() {
	// Create a kmd client
	kmdClient, err := kmd.MakeClient(kmdAddress, kmdToken)
	if err != nil {
		return
	}

	// Create an algod client
	algodClient, err := algod.MakeClient(algodAddress, algodToken)
	if err != nil {
		fmt.Printf("failed to make algod client: %s\n", err)
		return
	}

	// Create the wallet, if it doesn't already exist
	cwResponse, err := kmdClient.CreateWallet(walletName, walletPassword, kmd.DefaultWalletDriver, types.MasterDerivationKey{})
	if err != nil {
		fmt.Printf("error creating wallet: %s\n", err)
		return
	}

	// We need the wallet ID in order to get a wallet handle, so we can add accounts
	walletID := cwResponse.Wallet.ID
	fmt.Printf("Created wallet '%s' with ID: %s\n", cwResponse.Wallet.Name, walletID)

	// Get a wallet handle. The wallet handle is used for things like signing transactions
	// and creating accounts. Wallet handles do expire, but they can be renewed
	initResponse, err := kmdClient.InitWalletHandle(walletID, walletPassword)
	if err != nil {
		fmt.Printf("Error initializing wallet handle: %s\n", err)
		return
	}

	// Extract the wallet handle
	walletHandleToken := initResponse.WalletHandleToken

	// Generate a new address from the wallet handle
	genResponse, err := kmdClient.GenerateKey(walletHandleToken)
	if err != nil {
		fmt.Printf("Error generating key: %s\n", err)
		return
	}
	fmt.Printf("Generated address %s\n", genResponse.Address)

	// Extract the wallet address
	walletAddress := genResponse.Address

	// from the algorand community forum
	toAddress := "NJY27OQ2ZXK6OWBN44LE4K43TA2AV3DPILPYTHAJAMKIVZDWTEJKZJKO4A"

	// Wait for user input to confirm balance has been updated
	fmt.Printf("Time to get some ALGOS! Request some from https://bank.testnet.algorand.network/ for address: %s\n", walletAddress)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Have you requested the ALGOS? (Y/n): ")
	text, _ := reader.ReadString('\n')

	var account models.Account
	if text == "\n" || text == "Y\n" {
		i := 1
		fmt.Printf("checking if balance is greater than %d: attempt %d\n", amount, i)
		account, err = algodClient.AccountInformation(walletAddress)
		if err != nil {
			fmt.Printf("error getting account information: %s\n", err)
			return
		}
		for account.Amount < amount {
			i++
			time.Sleep(time.Duration(10 * 1000 * time.Millisecond))

			fmt.Printf("checking if balance is greater than %d: attempt %d\n", amount, i)
			account, err = algodClient.AccountInformation(walletAddress)
			if err != nil {
				fmt.Printf("error getting account information: %s\n", err)
				return
			}
		}
	} else {
		return
	}

	fmt.Printf("balance is greater than amount (%d). Current balance: %d\n", amount, account.Amount)
	fmt.Print("attempting to make transaction now\n")

	// Get the suggested transaction params
	txParams, err := algodClient.SuggestedParams()
	if err != nil {
		fmt.Printf("error getting suggested tx params: %s\n", err)
		return
	}

	// Make the transaction
	tx, err := transaction.MakePaymentTxn(walletAddress, toAddress, 1, amount, txParams.LastRound-1, txParams.LastRound, note, "", txParams.GenesisID)
	if err != nil {
		fmt.Printf("error creating transaction: %s\n", err)
		return
	}

	// Sign the transaction
	signResponse, err := kmdClient.SignTransaction(walletHandleToken, walletPassword, tx)
	if err != nil {
		fmt.Printf("failed to sign transaction with kmd: %s\n", err)
		return
	}
	fmt.Printf("kmd signed transaction with bytes: %x\n", signResponse.SignedTransaction)

	// Broadcast the transaction to the network
	sendResponse, err := algodClient.SendRawTransaction(signResponse.SignedTransaction)
	if err != nil {
		fmt.Printf("failed to send transaction: %s\n", err)
		return
	}

	fmt.Printf("Transaction has been broadcasted! ID: %s\n", sendResponse.TxID)
	fmt.Printf("View the transaction here - https://algoexplorer.io/tx/%s\n", sendResponse.TxID)
}
