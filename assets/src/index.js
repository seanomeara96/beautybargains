import "./htmx.min.js";
if (window.location.pathname.includes("products/")) {
    import("./timeline.js").then(res => {
        res.timeline()
    }).catch(console.log)
}

if(document.querySelector("#brand-description")){
    const container = document.getElementById("brand-description");
    container.style.overflow = "hidden"
    const button = document.getElementById("brand-description-read-more")
    let isOpen = false
    button.addEventListener("click", (e) => {
        e.preventDefault()
        container.classList.toggle("brand-description--is-faded")
        isOpen = !isOpen
        if(isOpen) {
            button.textContent = "Read Less"
        } else {
            button.textContent = "Read More"
        }
    })

}

