package api

import (
    "net/http"
    // "github.com/go-chi/chi/v5"
    "github.com/go-chi/render"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/go-playground/validator/v10"
    "log/slog"
    "nats_app/internal/services"
)

// we send only OrderId
type Request struct {
    OrderId string `json:"order_id"`
}

type Response struct {
    // Fields by default
    RespReport
}

// get order by id
func GetOrder(
    logger *slog.Logger,
    v *validator.Validate,
    ca *services.AppCache,
    ) http.HandlerFunc {
    return func(wr http.ResponseWriter, req *http.Request) {
        // const for idenfication
        const loc = "api.handlers"
        // setup call location & request_id for search
        logger = logger.With(
            slog.String("loc", loc),
            slog.String("request_id", middleware.GetReqID(req.Context())),
        )
        var request Request
        // parse requsest and set model
        err := render.DecodeJSON(req.Body, &request)
        if err != nil {
            logger.Error("Request decoding failed", err)
            render.JSON(wr, req, "can`t decode request")
            return
        }
        // in slog.any info about decoded request
        logger.Info("Request decoded", slog.Any("request", request))
        if val_err := v.Struct(request); val_err != nil {
            logger.Error("Invalid request", val_err)
            render.JSON(wr, req, "can`t parse request")
            return
        }
        // try fetch data from cache
        order, err := (*ca).Get(request.OrderId)
        if err != nil {
            logger.Error("Invalid order_id", err)
            render.JSON(wr, req, "order_id not found")
            return
        }
        _ = order
        logger.Info("order found.")
        render.JSON(wr, req, Response{
            RespReport: RespReport{},
            // Order: order,
        })
        return
    }
}
