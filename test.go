package main

import (
	"fmt"
	"net/http"
)

const testHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>List Tool Tester</title>
<style>
body{font-family:sans-serif;margin:2rem}
button{margin:0.5rem;}
</style>
</head>
<body>

<h1>List Tool Tester</h1>

<p>Test the list functions here.</p>

<button onclick="testOpen()">Test Open</button>
<button onclick="testClose()">Test Close</button>
<button onclick="testAdd()">Test Add</button>
<button onclick="testDelete()">Test Delete</button>
<button onclick="testList()">Test List</button>

<script>
async function testOpen() {
    const listName = prompt("Enter list name:");
    if (listName) {
        const res = await fetch('/open/' + encodeURIComponent(listName));
        const data = await res.text();
        alert(data);
    }
}

async function testClose() {
    const listName = prompt("Enter list name:");
    const index = prompt("Enter index (optional):");
    let url = '/close/' + encodeURIComponent(listName);
    if (index) {
        url += '?index=' + index;
    }
    const res = await fetch(url);
    const data = await res.text();
        alert(data);
}

async function testAdd() {
    const listName = prompt("Enter list name:");
    if (listName) {
        const res = await fetch('/add/' + encodeURIComponent(listName));
        const data = await res.text();
        alert(data);
    }
}

async function testDelete() {
    const listName = prompt("Enter list name:");
    if (listName) {
        const res = await fetch('/delete/' + encodeURIComponent(listName));
        const data = await res.text();
        alert(data);
    }
}

async function testList() {
    const listName = prompt("Enter list name:");
    if (listName) {
        const res = await fetch('/list/' + encodeURIComponent(listName));
        const data = await res.text();
        alert(data);
    }
}
</script>
</body></html>`

func (s *Store) handleTest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, testHTML)
}
