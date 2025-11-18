import { Application } from "https://cdn.skypack.dev/@hotwired/stimulus@3.2.2"
import TabsController from "./controllers/tabs_controller.js"
import UploadController from "./controllers/upload_controller.js"
import ReceiptsController from "./controllers/receipts_controller.js"
import ReimbursementsController from "./controllers/reimbursements_controller.js"
import ReimbursementDetailController from "./controllers/reimbursement_detail_controller.js"

const application = Application.start()
application.register("tabs", TabsController)
application.register("upload", UploadController)
application.register("receipts", ReceiptsController)
application.register("reimbursements", ReimbursementsController)
application.register("reimbursement-detail", ReimbursementDetailController)

// Debug: log when application is ready
console.log("Stimulus application started", application)

