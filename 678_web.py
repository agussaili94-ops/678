from flask import Flask, render_template_string, request, redirect, jsonify
import os, json

app = Flask(__name__)
HISTORY_FILE = 'history.json'

@app.route('/api/status')
def api_status():
    if os.path.exists('status.json'):
        try:
            with open('status.json', 'r') as f:
                return jsonify(json.load(f))
        except:
            pass
    return jsonify({})

@app.route('/handle', methods=['POST'])
def handle():
    action = request.form.get('act')
    rec_id = request.form.get('id', '').strip()
    url = request.form.get('url', '').strip()

    if action == "add" and rec_id and url:
        h = json.load(open(HISTORY_FILE)) if os.path.exists(HISTORY_FILE) else []
        h = [i for i in h if i['id'] != rec_id]
        h.insert(0, {"id": rec_id, "url": url})
        json.dump(h[:15], open(HISTORY_FILE, 'w'))
        os.system(f"tmux send-keys -t bigo.1 'start {rec_id}|{url}' C-m")
    elif action == "stop" and rec_id:
        os.system(f"tmux send-keys -t bigo.1 'stop {rec_id}' C-m")
    elif action == "del_hist" and rec_id:
        h = json.load(open(HISTORY_FILE)) if os.path.exists(HISTORY_FILE) else []
        h = [i for i in h if i['id'] != rec_id]
        json.dump(h, open(HISTORY_FILE, 'w'))

    return redirect('/')

@app.route('/')
def index():
    h = json.load(open(HISTORY_FILE)) if os.path.exists(HISTORY_FILE) else []
    return render_template_string(HTML, history=h)

HTML = '''
<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <style>
        :root { --primary: #1e5ba6; --bg: #f5f7fb; --white: #ffffff; }
        body { font-family: -apple-system, sans-serif; background: var(--bg); margin: 0; padding: 0; color: #333; line-height: 1.2; user-select: none; }
        .header { background: var(--primary); padding: 10px 15px; display: flex; justify-content: space-between; align-items: center; color: white; }
        .header h1 { font-size: 14px; margin: 0; font-weight: bold; text-transform: uppercase; }
        .container { padding: 10px; max-width: 480px; margin: auto; }
        .card { background: var(--white); border-radius: 8px; padding: 12px; margin-bottom: 10px; box-shadow: 0 1px 3px rgba(0,0,0,0.05); border: 1px solid #e1e4e8; }
        .input-form { display: flex; flex-direction: column; gap: 8px; }
        .input-form input { padding: 8px; border-radius: 6px; border: 1.5px solid #d1d5db; font-size: 13px; outline: none; }
        .btn-start { background: var(--primary); color: white; border: none; padding: 10px; border-radius: 6px; font-weight: bold; font-size: 12px; cursor: pointer; text-transform: uppercase; }
        .hist-chip { background: #e2e8f0; padding: 5px 10px; border-radius: 15px; font-size: 11px; cursor: pointer; font-weight: bold; display: inline-block; margin: 3px; color: #334155; }
        .status-header { background: #dee9f5; padding: 6px 12px; border-radius: 6px 6px 0 0; display: flex; justify-content: space-between; border: 1px solid #c8d6e5; border-bottom: none; font-size: 10px; font-weight: 900; color: #4a6fa5; }
        .session-container { background: var(--white); border: 1px solid #c8d6e5; border-radius: 0 0 6px 6px; padding: 8px 10px; }
        .mon-item { display: flex; flex-direction: column; padding: 10px 0; border-bottom: 1px solid #f1f5f9; gap: 6px; }
        .mon-item:last-child { border-bottom: none; }
        .mon-header { display: flex; justify-content: space-between; align-items: center; }
        .mon-nick { font-weight: 800; font-size: 13px; color: #1e293b; margin: 0; }
        .mon-status { font-size: 10px; font-weight: 900; margin: 0; display: flex; align-items: center; }
        .mon-metrics { display: flex; gap: 6px; font-size: 10px; color: #64748b; align-items: center; font-family: monospace; font-weight: bold; word-break: break-all; }
        .btn-stop { background: white; color: #dc2626; border: 1.2px solid #dc2626; padding: 4px 10px; border-radius: 6px; font-weight: 900; font-size: 9px; cursor: pointer; }

        @keyframes pulse { 0%, 100% { opacity: 1; transform: scale(1); } 50% { opacity: 0.3; transform: scale(1.1); } }
        .anim-pulse { animation: pulse 1s infinite; }
    </style>
</head>
<body>
    <div class="header">
        <h1>678 Web Recorder</h1>
        <span id="worker-count" style="font-size:10px; font-weight:bold; background:rgba(255,255,255,0.2); padding:2px 8px; border-radius:10px;">0/20</span>
    </div>
    <div class="container">
        <div class="card">
            <form action="/handle" method="POST" class="input-form">
                <input type="text" name="id" id="target_id" placeholder="Nama/Label Sesi (Contoh: Live1)" required autocomplete="off">
                <input type="text" name="url" id="target_url" placeholder="Tempel Link Stream 678..." required autocomplete="off">
                <button type="submit" name="act" value="add" class="btn-start">Mulai Rekam</button>
            </form>
        </div>
        <div class="card" style="padding: 8px 12px;">
            <div style="font-size:9px; font-weight:900; color:#94a3b8; text-transform:uppercase; margin-bottom:6px;">Riwayat Sesi (Klik untuk isi otomatis)</div>
            <div style="max-height: 80px; overflow-y: auto;">
                {% for h in history %}
                <span class="hist-chip" onclick="document.getElementById('target_id').value='{{h.id}}'; document.getElementById('target_url').value='{{h.url}}';" title="Klik untuk muat">{{ h.id }}</span>
                {% endfor %}
            </div>
        </div>
        <div class="status-header"><span>LIVE STATUS MONITOR</span><span>ACTIVE</span></div>
        <div class="session-container" id="session-list"></div>
    </div>

    <script>
        function stopWorker(id) {
            let fd = new FormData();
            fd.append('id', id); fd.append('act', 'stop');
            fetch('/handle', {method: 'POST', body: fd});
            let btn = document.getElementById('btn-stop-' + id);
            if(btn) { btn.innerText = 'WAIT...'; btn.style.background = '#e2e8f0'; btn.style.color = '#475569'; btn.style.border = 'none'; }
        }

        async function fetchState() {
            try {
                let res = await fetch('/api/status');
                let data = await res.json();
                let workers = Object.values(data);
                document.getElementById('worker-count').innerText = `${workers.length}/20`;
                let html = '';
                workers.forEach(w => {
                    let stCol = '#f97316'; let pulse = 'anim-pulse';
                    let speedDisplay = w.speed || '0 kb/s';

                    if (w.status.includes('RECORDING')) {
                        stCol = '#16a34a';
                    } else if (w.status.includes('REMUXING')) {
                        stCol = '#8e44ad'; speedDisplay = 'REMUX'; pulse = '';
                    } else if (w.status.includes('FINISHED')) {
                        stCol = '#3b82f6'; speedDisplay = 'DONE'; pulse = '';
                    }

                    html += `<div class="mon-item">
                        <div class="mon-header">
                            <span class="mon-nick">🏷️ ${w.id}</span>
                            <button type="button" id="btn-stop-${w.id}" onclick="stopWorker('${w.id}')" class="btn-stop" ${w.status.includes('REMUXING') || w.status.includes('FINISHED') ? 'style="display:none;"' : ''}>STOP</button>
                        </div>
                        <p class="mon-status" style="color:${stCol};">
                            <span style="font-size:12px; margin-right:4px;" class="${pulse}">●</span> ${w.status}
                        </p>
                        <div class="mon-metrics">
                            <span>🕒 ${w.duration}</span> |
                            <span>⚡ ${speedDisplay}</span> |
                            <span style="color:#3b82f6">💾 ${w.size || '0 MB'}</span>
                        </div>
                        <div style="font-size:9px; color:#64748b; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">🔗 ${w.url}</div>
                    </div>`;
                });
                document.getElementById('session-list').innerHTML = html || '<div style="text-align:center; padding:15px; font-size:11px; color:#94a3b8;">Belum ada rekaman aktif...</div>';
            } catch(e) {}
        }
        setInterval(fetchState, 1000);
        fetchState();
    </script>
</body>
</html>
'''

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080)
