const uploadArea = document.getElementById("upload-area");
const fileInput = document.getElementById("file-input");
const csrfToken = document.getElementById("csrf-token").value;

uploadArea.addEventListener("click", () => fileInput.click());

uploadArea.addEventListener("dragover", (e) => {
  e.preventDefault();
  uploadArea.classList.add("drag-over");
});

uploadArea.addEventListener("dragleave", () => {
  uploadArea.classList.remove("drag-over");
});

uploadArea.addEventListener("drop", (e) => {
  e.preventDefault();
  uploadArea.classList.remove("drag-over");
  handleFile(e.dataTransfer.files[0]);
});

fileInput.addEventListener("change", (e) => {
  handleFile(e.target.files[0]);
});

async function handleFile(file) {
  if (!file || !file.name.endsWith(".md")) {
    alert("Only .md files are allowed.");
    return;
  }

  const formData = new FormData();
  formData.append("file", file);

  try {
    const response = await fetch("/posts", {
      method: "POST",
      headers: {
        "X-CSRF-Token": csrfToken
      },
      body: formData
    });
    if (!response.ok) throw new Error("Upload failed");
    // alert("Upload successful");
  } catch (err) {
    console.error(err);
    // alert("Upload error");
  }
}