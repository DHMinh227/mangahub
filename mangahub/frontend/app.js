const API = "http://localhost:8080";

/* ------------------- LOAD MANGA LIST ------------------- */
async function loadManga() {
    const res = await fetch(API + "/manga");
    const data = await res.json();

    const table = document.getElementById("manga-list");
    if (!table) return;

    table.innerHTML = "";

    data.forEach(m => {
        const row = document.createElement("tr");
        row.innerHTML = `
            <td>${m.id}</td>
            <td><a href="manga.html?id=${m.id}">${m.title}</a></td>
            <td>${m.author}</td>
            <td>${m.status}</td>
            <td>${m.total_chapters}</td>
        `;
        table.appendChild(row);
    });
}
loadManga();

/* ------------------- SEARCH ------------------- */
async function searchManga() {
    const q = document.getElementById("search-input").value.toLowerCase();
    const res = await fetch(API + "/manga");
    const list = await res.json();

    const filtered = list.filter(m =>
        m.title.toLowerCase().includes(q) ||
        m.id.includes(q)
    );

    const table = document.getElementById("manga-list");
    table.innerHTML = "";

    filtered.forEach(m => {
        const row = document.createElement("tr");
        row.innerHTML = `
            <td>${m.id}</td>
            <td><a href="manga.html?id=${m.id}">${m.title}</a></td>
            <td>${m.author}</td>
            <td>${m.status}</td>
            <td>${m.total_chapters}</td>
        `;
        table.appendChild(row);
    });
}

/* ------------------- LOGIN ------------------- */
async function submitLogin() {
    const username = document.getElementById("login-username").value;
    const password = document.getElementById("login-password").value;

    const res = await fetch(API + "/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password })
    });

    const data = await res.json();

    if (data.access_token) {
        localStorage.setItem("token", data.access_token);
        localStorage.setItem("refresh_token", data.refresh_token);
        localStorage.setItem("username", username);

        window.location.href = "index.html";
    } else {
        alert("Login failed");
    }
}

/* ------------------- REGISTER ------------------- */
async function submitRegister() {
    const username = document.getElementById("reg-username").value;
    const password = document.getElementById("reg-password").value;

    const res = await fetch(API + "/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password })
    });

    const data = await res.json();

    if (data.access_token) {
        alert("Registered! Please log in.");
    } else {
        alert(data.error || "Registration failed");
    }
}

/* ------------------- LOGOUT ------------------- */
async function logout() {
    const refresh = localStorage.getItem("refresh_token");

    if (refresh) {
        await fetch(API + "/auth/logout", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ refresh_token: refresh })
        });
    }

    localStorage.clear();
    window.location.href = "index.html";
}

/* ------------------- AUTH UI ------------------- */
function updateAuthUI() {
    const username = localStorage.getItem("username");

    if (document.getElementById("auth-buttons")) {
        if (username) {
            document.getElementById("auth-buttons").style.display = "none";
            document.getElementById("logout-section").style.display = "block";
            document.getElementById("username-label").innerText =
                "Logged in as: " + username;
        } else {
            document.getElementById("auth-buttons").style.display = "block";
            document.getElementById("logout-section").style.display = "none";
        }
    }
}
updateAuthUI();

/* ------------------- MANGA DETAILS ------------------- */
async function loadMangaDetails() {
    const box = document.getElementById("title");
    if (!box) return; // Not on manga.html

    const params = new URLSearchParams(window.location.search);
    const id = params.get("id");

    const res = await fetch(API + "/manga/" + id);
    const m = await res.json();

    document.getElementById("title").innerText = m.title;
    document.getElementById("manga-id").innerText = m.id;
    document.getElementById("author").innerText = m.author;
    document.getElementById("genres").innerText = m.genres;
    document.getElementById("status").innerText = m.status;
    document.getElementById("chapters").innerText = m.total_chapters;
    document.getElementById("description").innerText = m.description;
}
loadMangaDetails();
