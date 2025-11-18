import { Controller } from "https://cdn.skypack.dev/@hotwired/stimulus@3.2.2"

export default class extends Controller {
    static targets = ["fileInput", "uploadBtn", "status", "progress", "progressFill", "progressText"]

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

                    const response = await fetch("/api/receipts", {
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

                    successCount++
                    uploaded = true
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

