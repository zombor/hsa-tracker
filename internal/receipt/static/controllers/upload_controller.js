import { Controller } from "https://cdn.skypack.dev/@hotwired/stimulus@3.2.2"

export default class extends Controller {
    static targets = ["fileInput", "uploadBtn", "status", "progress", "progressFill", "progressText", 
                      "modal", "receiptId", "receiptFilename", "receiptContentType", 
                      "receiptTitle", "receiptDate", "receiptAmount", "previewContainer"]

    connect() {
        console.log("Upload controller connected")
    }

    async submit(event) {
        event.preventDefault()
        console.log("Upload form submitted", event)

        const files = Array.from(this.fileInputTarget.files)
        if (files.length === 0) {
            this.showStatus("Please select at least one file", "error")
            return
        }

        this.uploadBtnTarget.disabled = true
        this.hideStatus()
        this.showProgress()

        let successCount = 0
        let errorCount = 0
        const errors = []

        // Process files sequentially to avoid overwhelming the server
        for (let i = 0; i < files.length; i++) {
            const file = files[i]
            this.updateProgress(i + 1, files.length, file.name)

            let retries = 0
            const maxRetries = 5
            let uploaded = false

            while (!uploaded && retries <= maxRetries) {
                try {
                    const formData = new FormData()
                    formData.append("file", file)

                    const response = await fetch("/api/receipts/scan", {
                        method: "POST",
                        body: formData
                    })

                    if (!response.ok) {
                        let errorMessage = "Upload failed"
                        let isRateLimit = response.status === 429
                        
                        try {
                            const errorData = await response.json()
                            if (errorData.error) {
                                errorMessage = errorData.error
                                // Check if error message indicates rate limit
                                if (errorMessage.includes("429") || 
                                    errorMessage.includes("quota") || 
                                    errorMessage.includes("rate limit") ||
                                    errorMessage.includes("Please retry")) {
                                    isRateLimit = true
                                }
                            }
                        } catch (e) {
                            errorMessage = response.statusText || "Upload failed"
                        }

                        // If rate limit error, try to parse retry time and wait
                        if (isRateLimit && retries < maxRetries) {
                            const retrySeconds = this.parseRetryTime(errorMessage)
                            if (retrySeconds > 0) {
                                const waitTime = Math.ceil(retrySeconds) + 1 // Add 1 second buffer
                                this.updateProgress(i + 1, files.length, `${file.name} (waiting ${waitTime}s...)`)
                                await this.sleep(waitTime * 1000)
                                retries++
                                continue
                            } else {
                                // If we can't parse retry time, use exponential backoff
                                const waitTime = Math.min(60, Math.pow(2, retries) * 5) // Max 60 seconds
                                this.updateProgress(i + 1, files.length, `${file.name} (retrying in ${waitTime}s...)`)
                                await this.sleep(waitTime * 1000)
                                retries++
                                continue
                            }
                        }

                        throw new Error(errorMessage)
                    }

                    const scanResult = await response.json()
                    
                    // Wait for user review
                    try {
                        await this.reviewReceipt(scanResult, file)
                        successCount++
                        uploaded = true
                    } catch (e) {
                        console.error("Review failed:", e)
                        throw new Error("User cancelled review")
                    }
                } catch (error) {
                    // If it's a rate limit error and we haven't exceeded max retries, continue loop
                    if ((error.message.includes("429") || 
                         error.message.includes("quota") || 
                         error.message.includes("rate limit") ||
                         error.message.includes("Please retry")) && 
                        retries < maxRetries) {
                        const retrySeconds = this.parseRetryTime(error.message)
                        if (retrySeconds > 0) {
                            const waitTime = Math.ceil(retrySeconds) + 1
                            this.updateProgress(i + 1, files.length, `${file.name} (waiting ${waitTime}s...)`)
                            await this.sleep(waitTime * 1000)
                            retries++
                            continue
                        } else {
                            const waitTime = Math.min(60, Math.pow(2, retries) * 5)
                            this.updateProgress(i + 1, files.length, `${file.name} (retrying in ${waitTime}s...)`)
                            await this.sleep(waitTime * 1000)
                            retries++
                            continue
                        }
                    }
                    
                    // If not rate limit or max retries exceeded, record error
                    errorCount++
                    errors.push(`${file.name}: ${error.message}`)
                    uploaded = true // Exit loop even on error
                }
            }
        }

        this.hideProgress()
        this.fileInputTarget.value = ""

        // Show summary
        if (errorCount === 0) {
            this.showStatus(
                `Successfully uploaded ${successCount} receipt${successCount > 1 ? 's' : ''}!`,
                "success"
            )
        } else if (successCount === 0) {
            this.showStatus(
                `Failed to upload ${errorCount} receipt${errorCount > 1 ? 's' : ''}. ${errors[0]}`,
                "error"
            )
        } else {
            this.showStatus(
                `Uploaded ${successCount} receipt${successCount > 1 ? 's' : ''}, ${errorCount} failed. ${errors[0]}`,
                "error"
            )
        }

        // Reload receipts after 2 seconds
        setTimeout(() => {
            this.hideStatus()
            // Trigger receipts reload via custom event
            window.dispatchEvent(new CustomEvent("receipts:reload"))
        }, 2000)

        this.uploadBtnTarget.disabled = false
    }

    reviewReceipt(data, file) {
        return new Promise((resolve, reject) => {
            this.reviewResolve = resolve
            this.reviewReject = reject

            this.receiptIdTarget.value = data.id
            this.receiptFilenameTarget.value = data.filename
            this.receiptContentTypeTarget.value = data.content_type
            this.receiptTitleTarget.value = data.title
            // Date needs to be YYYY-MM-DD for input[type=date]
            this.receiptDateTarget.value = data.date.split("T")[0]
            this.receiptAmountTarget.value = (data.amount / 100).toFixed(2)

            // Show preview
            this.previewTargetUrl = URL.createObjectURL(file)
            this.previewContainerTarget.innerHTML = ""
            
            if (file.type.startsWith("image/")) {
                const img = document.createElement("img")
                img.src = this.previewTargetUrl
                this.previewContainerTarget.appendChild(img)
            } else if (file.type === "application/pdf") {
                const embed = document.createElement("embed")
                embed.src = this.previewTargetUrl
                embed.type = "application/pdf"
                this.previewContainerTarget.appendChild(embed)
            } else {
                this.previewContainerTarget.textContent = "Preview not available"
            }

            this.modalTarget.style.display = "flex"
        })
    }

    async confirmReceipt(event) {
        event.preventDefault()
        
        const receiptData = {
            id: this.receiptIdTarget.value,
            filename: this.receiptFilenameTarget.value,
            content_type: this.receiptContentTypeTarget.value,
            title: this.receiptTitleTarget.value,
            date: new Date(this.receiptDateTarget.value).toISOString(),
            amount: Math.round(parseFloat(this.receiptAmountTarget.value) * 100)
        }

        try {
            const response = await fetch("/api/receipts", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(receiptData)
            })

            if (!response.ok) {
                throw new Error("Failed to save receipt")
            }

            const resolve = this.reviewResolve
            this.closeModal()
            if (resolve) resolve()
        } catch (error) {
            console.error("Error saving receipt:", error)
            alert("Failed to save receipt: " + error.message)
        }
    }

    cancelReview() {
        const reject = this.reviewReject
        this.closeModal()
        if (reject) reject(new Error("Cancelled by user"))
    }

    closeModal() {
        this.modalTarget.style.display = "none"
        this.reviewResolve = null
        this.reviewReject = null
        
        // Clean up preview URL
        if (this.previewTargetUrl) {
            URL.revokeObjectURL(this.previewTargetUrl)
            this.previewTargetUrl = null
        }
        this.previewContainerTarget.innerHTML = ""
    }

    showStatus(message, type) {
        this.statusTarget.textContent = message
        this.statusTarget.className = "status " + type
        this.statusTarget.style.display = "block"
    }

    hideStatus() {
        this.statusTarget.style.display = "none"
    }

    showProgress() {
        if (this.hasProgressTarget) {
            this.progressTarget.style.display = "block"
        }
    }

    hideProgress() {
        if (this.hasProgressTarget) {
            this.progressTarget.style.display = "none"
        }
    }

    updateProgress(current, total, filename) {
        if (!this.hasProgressFillTarget || !this.hasProgressTextTarget) {
            return
        }

        const percentage = (current / total) * 100
        this.progressFillTarget.style.width = percentage + "%"
        this.progressTextTarget.textContent = `${current} / ${total} - ${filename}`
    }

    parseRetryTime(errorMessage) {
        // Try to extract retry time from error message
        // Format: "Please retry in 50.404210191s."
        const match = errorMessage.match(/retry in ([\d.]+)s/i)
        if (match && match[1]) {
            return parseFloat(match[1])
        }
        return 0
    }

    sleep(ms) {
        return new Promise(resolve => setTimeout(resolve, ms))
    }
}

