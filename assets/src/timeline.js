import Chart from "chart.js/auto";
export function timeline() {
    const _productData = document.querySelector("#productdata")?.textContent
    if(!_productData) {
        console.log("no product data found");
        return
    }

    let productData
    try {
        productData = JSON.parse(_productData)
        if (typeof productData === "string") {
            productData = JSON.parse(productData)
        }
    } catch(err) {
        console.log(err)
        return
    }

    if(!productData) {
        console.log("no product data")
        return
    }

    console.log(typeof productData);
    console.log(productData.dates);
    console.log(productData.prices);

    const ctx = document.getElementById('price-chart');
    if (!ctx) {
        console.log("no chart element detected")
        return
    }

    new Chart(ctx, {
        type: 'line',
        data: {
            labels: productData.dates,
            datasets: [{
                label: 'Recorded Prices',
                data: productData.prices,
                borderWidth: 1
            }]
        },
        options: {
            scales: {
                y: {
                    beginAtZero: true
                }
            }
        }
    });

}
