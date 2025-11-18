import { Controller } from "https://cdn.skypack.dev/@hotwired/stimulus@3.2.2"

export default class extends Controller {
    static targets = ["tab", "content", "reimbursementDetail"]

    connect() {
        // Set initial active tab
        this.showTab("receipts")
    }

    switch(event) {
        event.preventDefault()
        const tabName = event.currentTarget.dataset.tabName
        this.showTab(tabName)
    }

    showTab(tabName) {
        // Hide all tabs and content
        this.tabTargets.forEach(tab => tab.classList.remove("active"))
        this.contentTargets.forEach(content => content.classList.remove("active"))
        
        // Hide reimbursement detail if visible
        if (this.hasReimbursementDetailTarget) {
            this.reimbursementDetailTarget.style.display = "none"
        }

        // Show selected tab
        const selectedTab = this.tabTargets.find(tab => tab.dataset.tabName === tabName)
        const selectedContent = this.contentTargets.find(content => content.dataset.tabName === tabName)
        
        if (selectedTab) selectedTab.classList.add("active")
        if (selectedContent) selectedContent.classList.add("active")

        // Clear receipt selection when switching tabs
        if (tabName === "receipts") {
            window.dispatchEvent(new CustomEvent("receipts:clearSelection"))
        }
    }
}

