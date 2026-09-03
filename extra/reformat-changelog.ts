import fs from "node:fs";

/**
 * Reformat the text produced by GitHub's generated release notes.
 *
 * Usage:
 *   npm run reformat-changelog -- release-notes.txt
 *   cat release-notes.txt | npm run reformat-changelog
 */
const file = process.argv[2];
const input = file ? fs.readFileSync(file, "utf8") : fs.readFileSync(0, "utf8");

const template = `
### 🆕 New Features
-

### ⬆️ Improvements
-

### 🐛 Bug Fixes
-

### 🔐 Security Fixes
-

### 📚 Documentation
-

### Others
- Other small changes, code refactoring and maintenance updates in wkarts/dockge.
`;

const lines = input.split("\n").map((line) => line.trim()).filter(Boolean);

for (const line of lines) {
    const byIndex = line.lastIndexOf(" by ");
    const inIndex = line.lastIndexOf(" in ");

    if (byIndex === -1 || inIndex === -1 || inIndex <= byIndex) {
        console.log(line.replace(/^\*\s*/, "- "));
        continue;
    }

    const message = line.slice(0, byIndex).replace(/^\*\s*/, "").trim();
    const username = line.slice(byIndex + 4, inIndex).trim();
    const pullRequestURL = line.slice(inIndex + 4).trim();
    const pullRequestID = pullRequestURL.match(/\/pull\/(\d+)(?:\D|$)/)?.[1];
    const pr = pullRequestID ? `#${pullRequestID}` : pullRequestURL;
    const thanks = username ? ` (Thanks ${username})` : "";

    console.log(`- ${pr} ${message}${thanks}`);
}

console.log(template);
