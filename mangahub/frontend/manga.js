const API = "http://localhost:8080";

async function loadMangaDetails() {
    const urlParams = new URLSearchParams(window.location.search);
    const mangaID = urlParams.get("id");

    const res = await fetch(API + "/manga/" + mangaID);
    const m = await res.json();

    document.getElementById("title").innerText = m.title;
    document.getElementById("id").innerText = m.id;
    document.getElementById("author").innerText = m.author;
    document.getElementById("genres").innerText = m.genres;
    document.getElementById("status").innerText = m.status;
    document.getElementById("chapters").innerText = m.total_chapters;
    document.getElementById("description").innerText = m.description;

    // show reading status only if logged in
    const token = localStorage.getItem("token");
    if (!token) return;

    // optional feature — we can activate later
}

loadMangaDetails();
