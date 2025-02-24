const fs = require("fs-extra");
const path = require("path");

const srcDir = path.join(__dirname, "..", "src", "views");
const destDir = path.join(__dirname, "..", "dist", "views");

fs.copySync(srcDir, destDir, { overwrite: true });
console.log("Views directory copied successfully");
