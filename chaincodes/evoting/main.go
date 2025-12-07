package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract struktur utama
type SmartContract struct {
	contractapi.Contract
}

// --- STRUKTUR DATA (STATE) ---

// Vote: Menyimpan data surat suara individual
type Vote struct {
	ReceiptID   string `json:"receiptId"`   // Token unik dari KPPS
	CandidateID string `json:"candidateId"` // ID Paslon yang dipilih
	Region      string `json:"region"`      // Daerah pemilihan (sesuai diagram ballot)
	Timestamp   string `json:"timestamp"`   // Waktu voting
}

// Candidate: Menyimpan data Paslon dan jumlah suara sementara
type Candidate struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"` // Menyimpan total suara (Fitur: Menghitung hasil)
}

// --- FUNGSI SMART CONTRACT ---

// 1. InitLedger: Mengatur aturan pemilihan & Kandidat awal
// Fungsi ini dijalankan sekali saat chaincode di-deploy untuk setup Paslon.
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	candidates := []Candidate{
		{ID: "01", Name: "Paslon Satu", Count: 0},
		{ID: "02", Name: "Paslon Dua", Count: 0},
	}

	for _, candidate := range candidates {
		candidateJSON, err := json.Marshal(candidate)
		if err != nil {
			return err
		}
		// Simpan state awal kandidat ke ledger
		err = ctx.GetStub().PutState(candidate.ID, candidateJSON)
		if err != nil {
			return fmt.Errorf("gagal menginisialisasi kandidat. %v", err)
		}
	}
	return nil
}

// 2. CastVote: Fungsi Inti (Validasi, Simpan State, Event Log)
func (s *SmartContract) CastVote(ctx contractapi.TransactionContextInterface, receiptId string, candidateId string, region string) error {
	
	// A. VALIDASI TRANSAKSI (Sesuai desain Anda)
	
	// Cek 1: Apakah token/receipt ini sudah pernah dipakai?
	voteCheck, _ := ctx.GetStub().GetState(receiptId)
	if voteCheck != nil {
		return fmt.Errorf("GAGAL: Token suara %s sudah digunakan!", receiptId)
	}

	// Cek 2: Apakah kandidat yang dipilih valid (ada di database)?
	candidateJSON, err := ctx.GetStub().GetState(candidateId)
	if err != nil || candidateJSON == nil {
		return fmt.Errorf("GAGAL: Kandidat %s tidak ditemukan!", candidateId)
	}

	// B. MENYIMPAN STATE PEMILU
	
	// 1. Rekam Suara Pemilih
	vote := Vote{
		ReceiptID:   receiptId,
		CandidateID: candidateId,
		Region:      region,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	voteJSON, _ := json.Marshal(vote)
	
	// Simpan Vote ke Ledger (Key: ReceiptID)
	err = ctx.GetStub().PutState(receiptId, voteJSON)
	if err != nil {
		return err
	}

	// 2. Update Jumlah Suara Kandidat (Real-time Counting)
	var candidate Candidate
	json.Unmarshal(candidateJSON, &candidate)
	candidate.Count = candidate.Count + 1 // Tambah 1 suara
	
	updatedCandidateJSON, _ := json.Marshal(candidate)
	// Simpan Kandidat yang sudah diupdate ke Ledger (Key: CandidateID)
	err = ctx.GetStub().PutState(candidateId, updatedCandidateJSON)
	if err != nil {
		return err
	}

	// C. MENGHASILKAN EVENT LOG
	// Event ini akan ditangkap oleh Backend/Dashboard secara real-time
	eventPayload := fmt.Sprintf("Suara baru masuk untuk %s dari wilayah %s", candidate.Name, region)
	ctx.GetStub().SetEvent("VoteCasted", []byte(eventPayload))

	return nil
}

// 3. GetVote: Transparansi & Audit (Cek status suara sendiri)
func (s *SmartContract) GetVote(ctx contractapi.TransactionContextInterface, receiptId string) (*Vote, error) {
	voteJSON, err := ctx.GetStub().GetState(receiptId)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca data: %v", err)
	}
	if voteJSON == nil {
		return nil, fmt.Errorf("suara tidak ditemukan")
	}

	var vote Vote
	json.Unmarshal(voteJSON, &vote)
	return &vote, nil
}

// 4. GetElectionResults: Menghitung Hasil Pemilu (Untuk Dashboard)
func (s *SmartContract) GetElectionResults(ctx contractapi.TransactionContextInterface) ([]*Candidate, error) {
	// Kita ambil data Paslon 01 dan 02
	// Dalam implementasi nyata, bisa menggunakan RangeQuery, tapi ini lebih efisien untuk jumlah paslon sedikit.
	ids := []string{"01", "02"}
	var results []*Candidate

	for _, id := range ids {
		candidateJSON, err := ctx.GetStub().GetState(id)
		if err != nil {
			return nil, err
		}
		var candidate Candidate
		json.Unmarshal(candidateJSON, &candidate)
		results = append(results, &candidate)
	}

	return results, nil
}

// 5. AuditLog: Melihat sejarah perubahan data (Opsional, fitur bawaan Fabric)
func (s *SmartContract) GetAssetHistory(ctx contractapi.TransactionContextInterface, key string) (string, error) {
	historyIterator, err := ctx.GetStub().GetHistoryForKey(key)
	if err != nil {
		return "", err
	}
	defer historyIterator.Close()

	// Logic untuk mem-parsing history bisa ditambahkan di sini
	// Untuk sekarang kita return success saja sebagai tanda fungsi ada
	return "History retrieved check ledger", nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		fmt.Printf("Error creating chaincode: %s", err.Error())
		return
	}
	chaincode.Start()
}