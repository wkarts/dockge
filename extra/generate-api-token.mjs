#!/usr/bin/env node
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';

const args = Object.fromEntries(process.argv.slice(2).map((v,i,a) => v.startsWith('--') ? [v.slice(2), a[i+1]?.startsWith('--') ? '' : a[i+1]] : null).filter(Boolean));
const name = args.name || 'automation';
const file = args.file || process.env.DOCKGE_API_TOKENS_FILE || './data/api-tokens.json';
const scopes = (args.scopes || 'server:read,stacks:read,stacks:write,stacks:delete,stacks:operate').split(',').map(v=>v.trim()).filter(Boolean);
const stackPrefixes = (args.prefixes || '').split(',').map(v=>v.trim()).filter(Boolean);
const token = crypto.randomBytes(32).toString('base64url');
const sha256 = crypto.createHash('sha256').update(token).digest('hex');
let doc = { tokens: [] };
if (fs.existsSync(file)) doc = JSON.parse(fs.readFileSync(file,'utf8'));
if (!Array.isArray(doc.tokens)) throw new Error('token file must contain tokens array');
if (doc.tokens.some(t => t.name === name && !t.disabled)) throw new Error(`active token name already exists: ${name}`);
doc.tokens.push({ name, sha256, scopes, stackPrefixes });
fs.mkdirSync(path.dirname(file), {recursive:true});
fs.writeFileSync(file, JSON.stringify(doc,null,2)+'\n', {mode:0o600});
try { fs.chmodSync(file,0o600); } catch {}
console.log('Dockge API token created. Copy this value now; it will not be stored in clear text again.');
console.log(token);
console.log(`Token file: ${file}`);
console.log(`Scopes: ${scopes.join(',')}`);
console.log(`Prefixes: ${stackPrefixes.join(',') || '(none - token cannot access stacks until configured)'}`);
