# GCP Service Account Identifier Manager

A tool to resolve service account ↔ subject ID mappings across GCP projects.

Find which service account belongs to a subject ID or verify which subject ID is associated with a service account email.

---

## 🛠 Why This Exists
This tool helps you:
- Find which service account corresponds to a **mysterious subject ID**.
- Verify the **actual subject ID** for a service account email.
- Audit which service accounts exist across projects and detect changes.

---

## 🔑 Key Features
### 🔍 Reverse Lookup
Find service account details by:
- **Subject ID (`uniqueId`)**
- **Service account email**
- **Project ID**

### 📊 CSV Export
- Stores all service accounts across projects.
- Tracks current status (active/deleted).
- Maintains project associations.

### 🚨 Change Tracking
- Detects when service accounts are **newly created**.
- Marks deleted service accounts in CSV.

---

## ✅ Requirements
- **Google Cloud SDK (`gcloud`)** authenticated.
- **Service Account Viewer permissions** on target projects.

**Note:** Generated CSVs are excluded from git (`.gitignore`) as they may contain sensitive information.

---

## ⚡ Usage

### 1️⃣ Run the Tool
```bash
$ go run main.go
```

You'll be prompted to choose a mode:
- **[L] Load service accounts** - Fetch service accounts from all accessible projects.
- **[A] Analyze existing data** - Search for service accounts based on email, subject ID, or project.

### 2️⃣ Analysis Mode
When selecting **Analyze Mode**, choose from:
- `[1]` Search by Project ID.
- `[2]` Search by Email.
- `[3]` Search Single Subject ID.
- `[4]` Bulk Subject ID Lookup.
- `[5]` Exit program.

### 3️⃣ Example Queries
#### Find service accounts in a project:
```bash
Enter Project ID: my-gcp-project-123
```

#### Find service account by email:
```bash
Enter partial email: my-service-account
```

#### Find service account by Subject ID:
```bash
Enter Subject ID: 123456789012345678901
```

#### Bulk lookup multiple Subject IDs:
```bash
Enter comma-separated Subject IDs: 123456789012345678901, 987654321098765432109
```

---

## 📄 CSV Output Format
The tool generates a CSV file named `service-accounts.csv` with the following fields:

| ProjectID       | Email                           | SubjectID                 | Status  |
|----------------|--------------------------------|---------------------------|---------|
| my-project-123 | my-sa@my-project.iam.gserviceaccount.com | 123456789012345678901 | active  |
| my-project-456 | another-sa@my-project.iam.gserviceaccount.com | 987654321098765432109 | deleted |

---

## ⚠️ Notes
- Ensure your **gcloud authentication** has access to the required projects.
- If projects are missing, verify permissions with `gcloud auth list` and `gcloud projects list`.
- Deleted service accounts remain in the CSV but are marked as **deleted**.
---

## 📌 License
This tool is open-source. Feel free to modify and contribute!

