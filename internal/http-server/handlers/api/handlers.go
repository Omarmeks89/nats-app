package api

import (
    "net/http"
    "log/slog"
    "encoding/json"
    "os"

    "github.com/go-chi/render"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/go-playground/validator/v10"

    "nats_app/internal/services"
    "nats_app/internal/storage"
)

// we send only OrderId
type Request struct {
    OrderId string `json:"order_uid"`
}

type Response struct {
    // Fields by default
    RespReport
    CustomerOrder storage.CustomerOrder
}

// get order by id
func GetOrder(
    v *validator.Validate,
    ca *services.AppCache,
    s services.AppStorage,
    ) http.HandlerFunc {
    return func(wr http.ResponseWriter, req *http.Request) {
        // const for idenfication
        const loc = "api.handlers"
        var cOrder storage.CustomerOrder
        // setup call location & request_id for search
        logger := slog.New(
            slog.NewTextHandler(
                os.Stdout,
                &slog.HandlerOptions{Level: slog.LevelDebug},
            ),
        )
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
            logger.Error("No same order", err)
            render.JSON(wr, req, "No same order")
            return
        }
        json.Unmarshal(*order.Payload, &cOrder)
        render.JSON(wr, req, Response{
            RespReport: RespReport{},
            CustomerOrder: cOrder,
        })
        return
    }
}
