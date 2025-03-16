package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Project struct {
	ProjectId string `json:"projectId"`
	Name      string `json:"name"`
}

type ServiceAccount struct {
	ProjectID string
	Email     string `json:"email"`
	UniqueId  string `json:"uniqueId"`
	Status    string
}

type AnalysisOptions struct {
	ProjectID string
	Email     string
	SubjectID string
}

type ProjectAccess struct {
	ProjectID string
	HasAccess string // "yes" or "no"
}

type projectResult struct {
	Project         Project
	ServiceAccounts []ServiceAccount
	Error           error
}

const maxConcurrent = 20 // Increased default concurrency

var concurrency int

func init() {
	flag.IntVar(&concurrency, "c", maxConcurrent, "Set concurrency level")
}

func main() {
	flag.Parse()
	fmt.Println("GCP Service Account Manager")
	fmt.Println("============================\n")

	fmt.Print("Choose mode:\n")
	fmt.Print("[L] Load service accounts\n")
	fmt.Print("[A] Analyze existing data\n")
	fmt.Print("Your choice (L/A): ")

	var modeChoice string
	fmt.Scanln(&modeChoice)

	if strings.ToUpper(modeChoice) == "A" {
		analyzeMode()
		return
	}

	// Original load mode
	projects, err := getGCPProjects()
	if err != nil {
		fmt.Printf("Error getting projects: %v\n", err)
		return
	}

	projectsToProcess, err := selectProjects(projects)
	if err != nil {
		fmt.Printf("Error selecting project: %v\n", err)
		return
	}

	existingAccounts := readExistingCSV()
	var accessStatuses []ProjectAccess
	processedProjects := concurrentProjectProcessing(projectsToProcess, existingAccounts, &accessStatuses)

	// Mark entries from processed projects as deleted if they no longer exist
	for key, acc := range existingAccounts {
		if processedProjects[acc.ProjectID] {
			if _, exists := existingAccounts[key]; !exists {
				acc.Status = "deleted"
				existingAccounts[key] = acc
			}
			existingAccounts[key] = acc
		}
	}

	if err := writeCSV(existingAccounts); err != nil {
		fmt.Printf("Error writing CSV: %v\n", err)
	}

	fmt.Println("Operation completed successfully")
	fmt.Printf(" - Service accounts: service-accounts.csv\n")
}

func analyzeMode() {
	for {
		fmt.Println("\nAnalysis Mode - Choose Action")
		fmt.Println("===============================")
		fmt.Print("[1] Search by Project ID\n")
		fmt.Print("[2] Search by Email\n")
		fmt.Print("[3] Search Single Subject ID\n")
		fmt.Print("[4] Bulk Subject ID Lookup\n")
		fmt.Print("[5] Exit program\n")
		fmt.Print("Your choice (1-5): ")

		var choice int
		_, err := fmt.Scanln(&choice)
		if err != nil || choice < 1 || choice > 5 {
			fmt.Println("Invalid input, please try again")
			continue
		}

		if choice == 5 {
			fmt.Println("Exiting...")
			os.Exit(0)
			break
		}

		accounts := readExistingCSVForAnalysis()

		switch choice {
		case 1:
			fmt.Print("Enter Project ID: ")
			var projectID string
			fmt.Scanln(&projectID)
			searchByProjectID(accounts, projectID)
		case 2:
			fmt.Print("Enter partial email: ")
			var email string
			fmt.Scanln(&email)
			searchByEmail(accounts, email)
		case 3:
			fmt.Print("Enter Subject ID: ")
			var subjectID string
			fmt.Scanln(&subjectID)
			searchBySubjectID(accounts, subjectID)
		case 4:
			fmt.Print("Enter comma-separated Subject IDs: ")
			var input string
			fmt.Scanln(&input)
			bulkSubjectIDLookup(accounts, input)
		}

		fmt.Print("\nPress Enter to continue...")
		fmt.Scanln() // Wait for user confirmation
	}
}

func searchByProjectID(accounts map[string]ServiceAccount, projectID string) {
	fmt.Printf("\nService accounts for project %s:\n", projectID)
	fmt.Println(strings.Repeat("-", 50))
	found := false

	for _, acc := range accounts {
		if acc.ProjectID == projectID {
			printAccount(acc)
			found = true
		}
	}

	if !found {
		fmt.Println("No matching accounts found")
	}
}

func searchByEmail(accounts map[string]ServiceAccount, email string) {
	fmt.Printf("\nSearch results for emails containing '%s':\n", email)
	fmt.Println(strings.Repeat("-", 50))
	found := false
	searchTerm := strings.ToLower(email)

	for _, acc := range accounts {
		if strings.Contains(strings.ToLower(acc.Email), searchTerm) {
			printAccount(acc)
			found = true
		}
	}

	if !found {
		fmt.Println("No accounts found with that email fragment")
	}
}

func searchBySubjectID(accounts map[string]ServiceAccount, subjectID string) {
	fmt.Printf("\nSearch results for Subject ID %s:\n", subjectID)
	fmt.Println(strings.Repeat("-", 50))
	found := false

	for _, acc := range accounts {
		if acc.UniqueId == subjectID {
			printAccount(acc)
			found = true
		}
	}

	if !found {
		fmt.Println("No matching accounts found")
	}
}

func bulkSubjectIDLookup(accounts map[string]ServiceAccount, input string) {
	subjectIDs := strings.Split(input, ",")
	var found []ServiceAccount
	var missing []string

	for _, rawID := range subjectIDs {
		subjectID := strings.TrimSpace(rawID)
		if subjectID == "" {
			continue
		}

		foundAccount := false
		for _, acc := range accounts {
			if acc.UniqueId == subjectID {
				found = append(found, acc)
				foundAccount = true
				break
			}
		}
		if !foundAccount {
			missing = append(missing, subjectID)
		}
	}

	printBulkResults(found, missing)
}

func printBulkResults(found []ServiceAccount, missing []string) {
	fmt.Println("\nBulk Subject ID Analysis")
	fmt.Println("========================")

	fmt.Printf("✅ Found %d accounts:\n", len(found))
	for _, acc := range found {
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("Subject ID:  %s\n", acc.UniqueId)
		fmt.Printf("Email:       %s\n", acc.Email)
		fmt.Printf("Project ID:  %s\n", acc.ProjectID)
		fmt.Printf("Status:      %s\n", acc.Status)
	}

	fmt.Printf("\n❌ Missing %d subject IDs:\n", len(missing))
	for _, id := range missing {
		fmt.Println(" -", id)
	}
}

func printAccount(acc ServiceAccount) {
	fmt.Printf("Project ID:  %s\n", acc.ProjectID)
	fmt.Printf("Email:       %s\n", acc.Email)
	fmt.Printf("Subject ID:  %s\n", acc.UniqueId)
	fmt.Printf("Status:      %s\n", acc.Status)
	fmt.Println(strings.Repeat("-", 50))
}

func getGCPProjects() ([]Project, error) {
	cmd := exec.Command("gcloud", "projects", "list", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gcloud command failed: %v", err)
	}

	var projects []Project
	err = json.Unmarshal(output, &projects)
	if err != nil {
		return nil, fmt.Errorf("failed to parse projects JSON: %v", err)
	}

	return projects, nil
}

func selectProjects(projects []Project) ([]Project, error) {
	fmt.Println("\nProcessing all available projects...")
	return projects, nil
}

func getServiceAccounts(projectID string) ([]ServiceAccount, error) {
	cmd := exec.Command("gcloud", "iam", "service-accounts", "list",
		"--project", projectID,
		"--format=json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %v", string(output), err)
	}

	var accounts []ServiceAccount
	err = json.Unmarshal(output, &accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service accounts JSON: %v", err)
	}

	return accounts, nil
}

func readExistingCSV() map[string]ServiceAccount {
	accounts := make(map[string]ServiceAccount)

	file, err := os.Open("service-accounts.csv")
	if err == nil {
		defer file.Close()
		reader := csv.NewReader(file)
		records, _ := reader.ReadAll()

		for i, row := range records {
			if i == 0 { // Skip header
				continue
			}
			if len(row) >= 4 {
				key := fmt.Sprintf("%s|%s|%s", row[0], row[1], row[2])
				accounts[key] = ServiceAccount{
					ProjectID: row[0],
					Email:     row[1],
					UniqueId:  row[2],
					Status:    row[3],
				}
			}
		}
	}
	return accounts
}

func readExistingCSVForAnalysis() map[string]ServiceAccount {
	accounts := make(map[string]ServiceAccount)

	file, err := os.Open("service-accounts.csv")
	if err != nil {
		fmt.Println("No existing data found - run load mode first")
		return accounts
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, _ := reader.ReadAll()
	for i, row := range records {
		if i == 0 || len(row) < 4 {
			continue
		}
		accounts[row[2]] = ServiceAccount{
			ProjectID: row[0],
			Email:     row[1],
			UniqueId:  row[2],
			Status:    row[3],
		}
	}

	return accounts
}

func writeCSV(accounts map[string]ServiceAccount) error {
	file, err := os.Create("service-accounts.csv")
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"ProjectID", "Email", "SubjectID", "Status"})
	for _, acc := range accounts {
		writer.Write([]string{acc.ProjectID, acc.Email, acc.UniqueId, acc.Status})
	}
	return nil
}

func concurrentProjectProcessing(projects []Project, existingAccounts map[string]ServiceAccount, accessStatuses *[]ProjectAccess) map[string]bool {
	start := time.Now()
	fmt.Printf("\n🚀 Processing %d projects with %d workers\n", len(projects), concurrency)
	progress := make(chan int, len(projects))

	// Progress reporter
	go func() {
		processed := 0
		for range progress {
			processed++
			fmt.Printf("\r📡 Progress: %d/%d projects (%.1f%%)", processed, len(projects), float64(processed)/float64(len(projects))*100)
		}
		fmt.Printf("\n✅ Completed in %.1f seconds\n", time.Since(start).Seconds())
	}()

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	results := make(chan projectResult)
	processed := make(map[string]bool)
	var mu sync.Mutex

	// Start workers
	for _, project := range projects {
		wg.Add(1)
		go func(p Project) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			accounts, err := getServiceAccounts(p.ProjectId)
			res := projectResult{
				Project:         p,
				ServiceAccounts: accounts,
				Error:           err,
			}
			results <- res
		}(project)
	}

	// Close results channel when all workers done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results
	for res := range results {
		mu.Lock()
		progress <- 1 // Update progress
		if res.Error != nil {
			*accessStatuses = append(*accessStatuses, ProjectAccess{
				ProjectID: res.Project.ProjectId,
				HasAccess: "no",
			})
		} else {
			*accessStatuses = append(*accessStatuses, ProjectAccess{
				ProjectID: res.Project.ProjectId,
				HasAccess: "yes",
			})
			processed[res.Project.ProjectId] = true

			for _, acc := range res.ServiceAccounts {
				key := fmt.Sprintf("%s|%s|%s", res.Project.ProjectId, acc.Email, acc.UniqueId)
				existingAccounts[key] = ServiceAccount{res.Project.ProjectId, acc.Email, acc.UniqueId, "active"}
			}
		}
		mu.Unlock()
	}

	// Add final newline after progress display
	fmt.Println()
	return processed
}
