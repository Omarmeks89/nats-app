async function getOrder() {
    const myForm = document.getElementById('ord_uid_form');
    const ordForm = document.getElementById('ord_data');
    
    // get data from field
    async function getFormValue(event) {
        event.preventDefault();

        // find wished form field
        const ordUID = myForm.querySelector('[name="ordField"]');
        // fetch value from them
        await loadDataFromServer(ordUID.value);
    }

    function clearForm(event) {
    }

    myForm.addEventListener('submit', getFormValue);
    myForm.addEventListener('reset', clearForm);
}

async function loadDataFromServer(uid) {
    var reqURL = `http://localhost:8000/orders`;
    console.log(reqURL);
    const responce = await fetch(reqURL, {
        method: "POST",
        headers: {
            Accept: "application/json",
            "Content-Type": "application/json",
            "User-Agent": "any-name",
            "Transfer-Encoding": "gzip",
        },
        body: JSON.stringify({"order_uid": uid}),
    });
    if (!responce.ok) {
        throw new Error(`Error on responce ${responce}`);
    }
    const result = await responce.json();
    await displayData(result);
}

async function displayData(data) {
    // need to stringify json
    for (key in data.CustomerOrder) {
        if ((typeof data.CustomerOrder[key]) == "object" ) {
            var nested = data.CustomerOrder[key];
            document.getElementById('ord_data').innerHTML += 
                `<b><code>${key}:</code></b><br>`;
            for (objKey in nested) {
                // console.log(JSON.stringify(nested[objKey]));
                if ((typeof nested[objKey]) == "object") {
                    var item = nested[objKey];
                    for (k in item) {
                        document.getElementById('ord_data').innerHTML += 
                            `<code>**${k}: ${item[k]}</code><br>`;
                    }
                } else {
                document.getElementById('ord_data').innerHTML += 
                    `<code>*${objKey}: ${nested[objKey]}</code><br>`;
                }
            }
        }
        else { 
            document.getElementById('ord_data').innerHTML += 
                `<code>${key}: ${data.CustomerOrder[key]}</code><br>`;
        }
    }
}

getOrder();
