const API = "http://localhost:8080";

async function submitLogin() {
    let username = document.getElementById("login-username").value;
    let password = document.getElementById("login-password").value;

    const res = await fetch(API + "/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password })
    });

    const data = await res.json();

    if (data.token) {
        localStorage.setItem("token", data.token);
        localStorage.setItem("username", username);
        alert("Login successful!");
        window.location.href = "index.html";
    } else {
        alert("Login failed");
    }
}

async function submitRegister() {
    let username = document.getElementById("reg-username").value;
    let password = document.getElementById("reg-password").value;

    const res = await fetch(API + "/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password })
    });

    const data = await res.json();

    alert(data.message || JSON.stringify(data));
}
