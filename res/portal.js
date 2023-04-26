const validAddress = /[A-Z0-9]{12}/

document.addEventListener("DOMContentLoaded", (evt)=>{
    document.querySelectorAll("button.glyph").forEach( (emt) => {
        emt.addEventListener("click",clickButton)
    })
    document.querySelector("button#clear").addEventListener("click", (evt)=>{
        document.querySelector("#hexaddress").value = ""
        document.querySelector("#url").value = ""
        document.querySelector("#address").classList.add("hidden")
        validate()
    })
    document.querySelector("button#load").addEventListener("click", loadAddress)
})

function clickButton(evt) {
    console.log("clicked button", evt.target.value)
    appendSymbol(evt.target.value)
}

function appendSymbol(symbol) {
    let hexaddressField = document.querySelector("#hexaddress")
    let newValue = hexaddressField.value + symbol
    if (newValue.length > 12) {
        newValue = newValue.slice(-12)
    }
    hexaddressField.value = newValue
    validate()
}

function validate() {
    let value = document.querySelector("#hexaddress").value
    let loadButton = document.querySelector("button#load")
    if (value.match(validAddress)){
        loadButton.classList.remove("nope")
        loadButton.disabled = false
        return true
    }
    else {
        loadButton.classList.add("nope")
        loadButton.disabled = true
        return false
    }
}

function loadAddress() {
    if (!validate()) {
        return
    }
    let address = document.querySelector("#hexaddress").value
    let url = host = window.location.protocol + "//" + window.location.host + "/" + address + ".png"
    document.querySelector("#url").value = url
    let image = document.querySelector("#address")
    image.src = url
    image.classList.remove("hidden")
}