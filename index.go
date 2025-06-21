
package main

/* --------------------------------------------------------------------- */
/* 2.  Embedded index.html                                               */
/*   yeah...i put it here...                                              */
/* --------------------------------------------------------------------- */

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Review-Board</title>
<style>
body{font-family:sans-serif;margin:2rem}
table{border-collapse:collapse;width:100%;margin-bottom:2rem}
th,td{border:1px solid #ddd;padding:.4rem;text-align:left}
tr:hover{background:#f3f3f3}
.badge{padding:2px 6px;border-radius:4px;color:#fff;font-size:.8rem}
.open{background:#28a745}.closed{background:#6c757d}
form{margin-top:1rem}
</style>
</head>
<body>


<h1>OpenHands MCP List Tool</h1>

<!-- Route cheat-sheet -->
<a href="/test">Test List Functions</a><br><br><table class="routes">
<thead><tr><th>Verb & Path</th><th>Description</th></tr></thead>
<tbody>
<tr><td><code>GET /open/{list}</code></td>   <td>Return first open item with its index</td></tr>
<tr><td><code>GET /close/{list}?index=n</code></td><td>Close item (<code>index</code> optional)</td></tr>
<tr><td><code>GET /add/{list}</code></td>    <td>Create empty list</td></tr>
<tr><td><code>POST /add/{list}</code></td>   <td>Create list, seed with JSON array</td></tr>
<tr><td><code>GET /delete/{list}</code></td> <td>Delete list</td></tr>
<tr><td><code>GET /list/{list}</code></td>   <td>Return full list (JSON)</td></tr>
<tr><td><code>GET /timeout/{seconds}</code></td><td>Set throttle delay (0-600 s)</td></tr>
<tr><td><code>GET /meta</code></td>          <td>Summary for index page</td></tr>
<tr><td><code>/ or /index.html</code></td>   <td>This web UI</td></tr>
</tbody>
</table>

<table id="lists">
<thead><tr><th>Name</th><th>Total</th><th>Open</th></tr></thead>
<tbody></tbody></table>

<h2>Throttle Open / Close</h2>
<div>
<p> The main function of the throttle is to slow down the AI tool</p>
<p> This way you can use gemini without running immediately into limits</p>
</div>
<form id="delayForm">
<label>Delay&nbsp;(seconds):
  <input id="delaySeconds" type="number" min="0" max="600" value="0" required>
</label>
<button type="submit">Set&nbsp;delay</button>
<span id="currentDelay" style="margin-left:1rem;color:#555"></span>
</form>

<h2>Add / Seed List</h2>
<form id="seedForm">
<label>List&nbsp;name:
  <input id="listName" required>
</label><br><br>
<label>JSON&nbsp;array&nbsp;of&nbsp;items:<br>
  <textarea id="jsonBody" rows="10" cols="80"
   placeholder='[{"Document":"a.md","conflict":"x","new_statement":"y"}]'></textarea>
</label><br><br>
<button type="submit">POST /add/{list}</button>
</form>

<script>
async function refresh(){
  const res=await fetch('/meta');
  const data=await res.json();

  // update list table
  const tbody=document.querySelector('#lists tbody');
  tbody.innerHTML='';
  data.lists.forEach(l=>{
    const tr=document.createElement('tr');
    tr.innerHTML=
      '<td>'+l.name+'</td>'+
      '<td>'+l.count+'</td>'+
      '<td><span class="badge '+(l.open? "open":"closed")+'">'+l.open+'</span></td>';
    tbody.appendChild(tr);
  });

  // update delay display
  document.getElementById('currentDelay').textContent='current: '+data.delay+' s';
  document.getElementById('delaySeconds').value=data.delay;
}

document.getElementById('delayForm').addEventListener('submit',async e=>{
  e.preventDefault();
  const secs=document.getElementById('delaySeconds').value;
  try{
    const res=await fetch('/timeout/'+secs);
    if(!res.ok) throw new Error(await res.text());
    refresh();
  }catch(err){alert(err);}
});

document.getElementById('seedForm').addEventListener('submit',async e=>{
  e.preventDefault();
  const name=document.getElementById('listName').value.trim();
  const body=document.getElementById('jsonBody').value.trim()||'[]';
  try{
    const res=await fetch('/add/'+encodeURIComponent(name),{
      method:'POST',
      headers:{'Content-Type':'application/json'},
      body:body
    });
    if(!res.ok) throw new Error(await res.text());
    alert('Success!');
    document.getElementById('jsonBody').value='';
    refresh();
  }catch(err){alert(err);}
});

refresh(); setInterval(refresh,5000);
</script>
</body></html>`
