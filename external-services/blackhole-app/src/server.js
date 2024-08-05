import express from "express";
import httpProxy from 'http-proxy';
import bodyParser from 'body-parser'

const app = express();
const PORT = 8080;

const { createProxyServer } = httpProxy
const proxy = createProxyServer({});

const namespace = process.env.NAMESPACE ?? "chaosmania"

app.get("/", (req, res) => {
    res.send("Hello from Express!");
});

function logAccess(path) {
    console.info(`proxy-pass: ${path}`)
}

app.post("/recommends", (req, res) => {
    logAccess("/recommends")
    proxy.web(req, res, {target: `http://recommendation.${namespace}/`})
});

app.post("/prodcat", (req, res) => {
    logAccess("/prodcat")
    proxy.web(req, res, {target: `http://productcatalog.${namespace}/`})
});

app.post("/shipment", (req, res) => {
    logAccess("/shipment")
    proxy.web(req, res, {target: `http://shipping.${namespace}/`})
});

app.listen(PORT, () => {
    console.log(`Express server running at http://localhost:${PORT}/`);
});

// proxy.on('proxyReq', function(proxyReq, req, res, options) {
//     console.log(`proxyReq Method = ${proxyReq.method}`)
//     proxyReq.method = 'POST';
// });