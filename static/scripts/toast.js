// Minimal toast notifications. Inline field errors are preferred for anything
// the user can fix in place; toasts are for job outcomes and server problems.

const MAX_TOASTS = 3;

export function toast(message, { type = "info", timeout = 5000 } = {}) {
    const container = document.getElementById("toasts");
    if (!container) return;

    const el = document.createElement("div");
    el.className = `toast toast-${type}`;
    el.setAttribute("role", type === "error" ? "alert" : "status");

    const text = document.createElement("p");
    text.textContent = message;

    const close = document.createElement("button");
    close.type = "button";
    close.className = "toast-close";
    close.setAttribute("aria-label", "Dismiss");
    close.textContent = "×";

    const remove = () => el.remove();
    close.addEventListener("click", remove);
    el.append(text, close);
    container.append(el);

    while (container.children.length > MAX_TOASTS) container.firstElementChild.remove();
    if (timeout > 0) setTimeout(remove, timeout);
}
