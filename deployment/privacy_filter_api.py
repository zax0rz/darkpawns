from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional
import privacy_filter
import uvicorn

app = FastAPI(title="OpenAI Privacy Filter API", version="1.0.0")

# PII categories
PII_CATEGORIES = [
    "account_number",
    "address", 
    "email",
    "person",
    "phone",
    "url",
    "date",
    "secret"
]

class FilterConfig(BaseModel):
    categories: Optional[List[str]] = None
    replacement: str = "[REDACTED]"
    keep_length: bool = False

class FilterRequest(BaseModel):
    text: str
    config: FilterConfig

class FilterResponse(BaseModel):
    filtered_text: str
    detected_categories: List[str]
    error: Optional[str] = None

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy"}

@app.get("/categories")
async def get_categories():
    """Get available PII categories"""
    return {"categories": PII_CATEGORIES}

@app.post("/filter", response_model=FilterResponse)
async def filter_text(request: FilterRequest):
    """Filter PII from text"""
    try:
        # Use all categories if none specified
        categories = request.config.categories or PII_CATEGORIES
        
        # Convert categories to privacy-filter format
        filter_categories = []
        for cat in categories:
            if cat == "account_number":
                filter_categories.append(privacy_filter.AccountNumber)
            elif cat == "address":
                filter_categories.append(privacy_filter.Address)
            elif cat == "email":
                filter_categories.append(privacy_filter.Email)
            elif cat == "person":
                filter_categories.append(privacy_filter.Person)
            elif cat == "phone":
                filter_categories.append(privacy_filter.Phone)
            elif cat == "url":
                filter_categories.append(privacy_filter.URL)
            elif cat == "date":
                filter_categories.append(privacy_filter.Date)
            elif cat == "secret":
                filter_categories.append(privacy_filter.Secret)
            else:
                return FilterResponse(
                    filtered_text=request.text,
                    detected_categories=[],
                    error=f"Unknown category: {cat}"
                )
        
        # Apply privacy filter
        filtered_text, detected = privacy_filter.filter(
            request.text,
            categories=filter_categories,
            replacement=request.config.replacement,
            keep_length=request.config.keep_length
        )
        
        # Convert detected categories back to string names
        detected_names = []
        for det in detected:
            if det == privacy_filter.AccountNumber:
                detected_names.append("account_number")
            elif det == privacy_filter.Address:
                detected_names.append("address")
            elif det == privacy_filter.Email:
                detected_names.append("email")
            elif det == privacy_filter.Person:
                detected_names.append("person")
            elif det == privacy_filter.Phone:
                detected_names.append("phone")
            elif det == privacy_filter.URL:
                detected_names.append("url")
            elif det == privacy_filter.Date:
                detected_names.append("date")
            elif det == privacy_filter.Secret:
                detected_names.append("secret")
        
        return FilterResponse(
            filtered_text=filtered_text,
            detected_categories=detected_names
        )
        
    except Exception as e:
        return FilterResponse(
            filtered_text=request.text,
            detected_categories=[],
            error=str(e)
        )

@app.post("/batch_filter")
async def batch_filter(requests: List[FilterRequest]):
    """Filter multiple texts at once"""
    responses = []
    for req in requests:
        response = await filter_text(req)
        responses.append(response)
    return responses

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)