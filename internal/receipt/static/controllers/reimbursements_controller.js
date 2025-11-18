import { Controller } from "https://cdn.skypack.dev/@hotwired/stimulus@3.2.2"

export default class extends Controller {
    static targets = ["container"]

    connect() {
        this.load()
        
        // Listen for reload events
        this.reloadHandler = () => this.load()
        window.addEventListener("reimbursements:reload", this.reloadHandler)
    }
    
    disconnect() {
        window.removeEventListener("reimbursements:reload", this.reloadHandler)
    }

    async load() {
        try {
            const response = await fetch("/api/reimbursements")
            if (!response.ok) throw new Error("Failed to load reimbursements")

            const reimbursements = await response.json()
            this.display(reimbursements)
        } catch (error) {
            this.containerTarget.innerHTML = 
                '<div class="empty-state">Error loading reimbursements: ' + error.message + "</div>"
        }
    }

    display(reimbursements) {
        if (!reimbursements || reimbursements.length === 0) {
            this.containerTarget.innerHTML = '<div class="empty-state">No reimbursements yet.</div>'
            return
        }

        // Sort by date (newest first)
        reimbursements.sort((a, b) => new Date(b.created_at) - new Date(a.created_at))

        this.containerTarget.innerHTML = reimbursements.map(reimbursement => {
            const date = new Date(reimbursement.created_at).toLocaleDateString()
            const amount = "$" + (reimbursement.total_amount / 100).toFixed(2)
            const receiptCount = reimbursement.receipt_ids.length

            return `<div class="reimbursement-item">
                <div class="reimbursement-info">
                    <div class="reimbursement-title">Reimbursement #${this.escapeHtml(reimbursement.id.substring(0, 8))}</div>
                    <div class="reimbursement-meta">${date} • ${receiptCount} receipt${receiptCount > 1 ? "s" : ""}</div>
                </div>
                <div style="display: flex; align-items: center;">
                    <span class="reimbursement-amount">${amount}</span>
                    <div class="reimbursement-actions">
                        <button class="btn-small" data-action="click->reimbursements#view" data-reimbursement-id="${this.escapeHtml(reimbursement.id)}">View</button>
                    </div>
                </div>
            </div>`
        }).join("")
    }

    async view(event) {
        const reimbursementId = event.currentTarget.dataset.reimbursementId
        console.log("View reimbursement clicked", reimbursementId)
        // Trigger detail view via custom event
        const customEvent = new CustomEvent("reimbursement-detail:show", { detail: { id: reimbursementId } })
        console.log("Dispatching event", customEvent)
        window.dispatchEvent(customEvent)
    }

    escapeHtml(text) {
        const div = document.createElement("div")
        div.textContent = text
        return div.innerHTML
    }
}

