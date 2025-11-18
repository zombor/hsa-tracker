# HSA Tracker

A web-based application for tracking Health Savings Account (HSA) expenses. Upload receipts, automatically extract expense details using AI, and track reimbursements—all from your phone or computer.

## Features

- 📸 **Upload Receipts**: Upload images (JPG, PNG) or PDFs from your phone or computer
- 🤖 **AI-Powered Scanning**: Automatically extracts store name, date, and amount from receipts using Google Gemini or Ollama
- 💾 **Local Storage**: All receipts and data stored locally on your machine
- 📱 **Mobile-Friendly**: Web interface optimized for taking photos on your phone
- 💰 **Expense Tracking**: View total receipts and total value at a glance
- ✅ **Reimbursement Tracking**: Mark receipts as reimbursed and track reimbursement events
- 🔒 **Optional Authentication**: Basic auth protection for your data

## Installation

### Prerequisites

- Go 1.24 or later
- Google Gemini API key (for cloud-based scanning) OR Ollama installed locally (for local scanning)

### Build

```bash
git clone <repository-url>
cd hsa-tracker
go build -o hsa-tracker ./cmd/hsa-tracker
```

## Usage

### Quick Start

1. **Set your Gemini API key** (if using Gemini):
   ```bash
   export GEMINI_API_KEY=your-api-key-here
   ```

2. **Run the server**:
   ```bash
   ./hsa-tracker
   ```

3. **Open your browser**:
   Navigate to `http://localhost:8080`

### Configuration Options

All configuration can be done via command-line flags or environment variables (prefixed with `HSA_TRACKER_`):

#### Basic Options

- `--port` (default: `8080`): HTTP server port
- `--db` (default: `hsa-tracker.db`): Path to the database file
- `--storage` (default: `./receipts`): Directory where receipt files are stored

#### Scanner Options

- `--scanner` (default: `gemini`): Scanner type - `gemini` or `ollama`
- `--gemini-key`: Google Gemini API key (or set `GEMINI_API_KEY` env var)
- `--gemini-model` (default: `gemini-2.5-pro`): Gemini model to use
- `--ollama-url` (default: `http://localhost:11434`): Ollama API URL
- `--ollama-model` (default: `llava`): Ollama model name (e.g., `llava`, `llava-phi3`, `bakllava`, `qwen2-vl`)

#### Security Options

- `--auth-user`: Basic auth username (optional)
- `--auth-pass`: Basic auth password (optional)

### Example: Using Gemini (Cloud)

```bash
export GEMINI_API_KEY=your-api-key-here
./hsa-tracker --port 8080 --storage ~/hsa-receipts
```

### Example: Using Ollama (Local)

```bash
# Make sure Ollama is running with a vision model installed
# e.g., ollama pull llava

./hsa-tracker --scanner ollama --ollama-model llava --port 8080
```

### Example: With Authentication

```bash
./hsa-tracker --auth-user myuser --auth-pass mypassword
```

### Using the Web Interface

1. **Upload Receipts**:
   - Click "Upload Receipts" or use the file input
   - Select one or multiple receipt images/PDFs
   - The app will automatically scan and extract details
   - Progress is shown during bulk uploads

2. **View Receipts**:
   - All receipts are listed on the main page, sorted by date (newest first)
   - View total receipt count and total value at the top
   - Click "View" to see the original receipt file
   - Click "Delete" to remove a receipt (only if not reimbursed)

3. **Mark as Reimbursed**:
   - Select one or more receipts using the checkboxes
   - Click "Mark as Reimbursed" in the selection bar
   - This creates a reimbursement event linking all selected receipts

4. **View Reimbursements**:
   - Switch to the "Reimbursements" tab
   - See a list of all reimbursement events
   - Click on a reimbursement to see details and associated receipts

### Data Storage

- **Database**: Receipt metadata is stored in `hsa-tracker.db` (BoltDB)
- **Files**: Original receipt files are stored in the `--storage` directory (default: `./receipts`)
- Both are stored locally on your machine—your data never leaves your control

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./internal/receipt/...
```

### Project Structure

```
hsa-tracker/
├── cmd/hsa-tracker/     # Main application entry point
├── internal/
│   ├── receipt/          # Receipt domain logic (DB, storage, service, handlers)
│   └── scanning/         # LLM scanning abstraction (Gemini, Ollama)
└── go.mod                # Go module dependencies
```

### Key Technologies

- **Go 1.23+**: Core language
- **BoltDB**: Embedded key-value database
- **Google Gemini API**: Cloud-based LLM for receipt scanning
- **Ollama**: Local LLM option for receipt scanning
- **Ginkgo/Gomega**: BDD testing framework
- **Stimulus.js**: Frontend JavaScript framework
- **Tailwind CSS**: Styling

### Adding New Features

- **New LLM Provider**: Implement the `scanning.Scanner` interface
- **New Storage Backend**: Implement the `receipt.Storage` interface
- **New Database Backend**: Implement the `receipt.DB` interface

### Testing Conventions

- Use Ginkgo/Gomega for all tests
- Use `When()` blocks instead of `Context()`
- Set up state in `BeforeEach`, execute in `JustBeforeEach`
- One logical assertion per `It` block
- Use `MatchError()` for error assertions
- Wrap errors with `fmt.Errorf()` in production code

## License

This project is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).

See the [LICENSE](LICENSE) file for details.
