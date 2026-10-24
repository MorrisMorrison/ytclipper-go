// Light/dark theme. The inline script in index.html sets the initial theme
// before first paint (stored choice, else prefers-color-scheme).

const root = document.documentElement;

function current() {
    return root.getAttribute("data-theme") === "dark" ? "dark" : "light";
}

function apply(theme, button) {
    root.setAttribute("data-theme", theme);
    button.setAttribute("aria-pressed", String(theme === "dark"));
}

export function initTheme(button) {
    apply(current(), button);

    button.addEventListener("click", () => {
        const next = current() === "dark" ? "light" : "dark";
        apply(next, button);
        try {
            localStorage.setItem("theme", next);
        } catch {
            // storage blocked: the choice lasts for this page only
        }
    });

    // Follow the OS while the user has not picked a theme.
    window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", (event) => {
        let stored = null;
        try {
            stored = localStorage.getItem("theme");
        } catch {
            // ignore
        }
        if (stored !== "light" && stored !== "dark") apply(event.matches ? "dark" : "light", button);
    });
}
