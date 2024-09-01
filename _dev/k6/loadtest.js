import http from "k6/http";
import { check } from "k6";

export const options = {
    vus: 1,
    duration: "10s",
    thresholds: {
        http_req_duration: ["p(95)<900"],
    },
};

export function setup() {
    console.log("Starting the test");
}

export default function () {
    const response = http.get("http://app:8000/wait");
    check(response, {
        "status is 200": (r) => r.status === 200,
    });
}
