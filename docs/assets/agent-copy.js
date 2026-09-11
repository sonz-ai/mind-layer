const button = document.getElementById("copy-agent-prompt");
const prompt = document.getElementById("agent-prompt");
const status = document.getElementById("agent-copy-status");

button?.addEventListener("click", async () => {
  try {
    await navigator.clipboard.writeText(prompt.value);
    button.textContent = "Copy again";
    status.textContent = "Copied. Paste into ChatGPT, Claude, or your coding agent.";
  } catch {
    prompt.focus();
    prompt.select();
    status.textContent = "Copy was blocked. The instruction is selected; use your browser’s Copy command.";
  }
});
